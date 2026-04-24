// Package amountfast is an int64 fixed-point monetary type optimized for
// high-throughput simulation (e.g., RTP/slot spin workloads). Scale is 10^6
// (6 decimal places). Arithmetic ops are zero-alloc.
//
// Mul/Div truncate toward zero. Use the Round* methods for explicit rounding.
// For production money handling with arbitrary precision, use the "amount"
// package instead.
package amountfast

import (
	"database/sql/driver"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type Value struct {
	scaled int64
	raw    string
}

const (
	Scale    = 1_000_000
	scaleF64 = float64(Scale)
)

var Zero = Value{}

// FromFloat rounds to the nearest scaled unit to avoid repeating-decimal
// drift (e.g., 0.1 * 1e6 = 99999.99999... without rounding).
func FromFloat(f float64) Value     { return Value{scaled: int64(math.Round(f * scaleF64))} }
func FromInt(i int) Value           { return Value{scaled: int64(i) * Scale} }
func FromInt64(i int64) Value       { return Value{scaled: i * Scale} }
func FromInt32(i int32) Value       { return Value{scaled: int64(i) * Scale} }
func FromScaled(scaled int64) Value { return Value{scaled: scaled} }

func FromString(s string) (Value, error) {
	var v Value
	if err := v.setFromString(s); err != nil {
		return Value{}, err
	}
	return v, nil
}

// FromDecimal converts via decimal's own precision (no float64 round-trip).
// Truncates beyond 6 decimal places toward zero.
func FromDecimal(d decimal.Decimal) Value {
	return Value{scaled: d.Shift(6).Truncate(0).IntPart()}
}

func (m Value) Float64() float64 { return float64(m.scaled) / scaleF64 }
func (m Value) Int64() int64     { return m.scaled / Scale }
func (m Value) Int32() int32     { return int32(m.scaled / Scale) }
func (m Value) Int() int         { return int(m.scaled / Scale) }

// Scaled returns the raw int64 scaled by 10^6.
func (m Value) Scaled() int64 { return m.scaled }

// Raw returns the original input string (set via FromString/Scan/Unmarshal*).
// Empty for values constructed from numeric types.
func (m Value) Raw() string { return m.raw }

func (m Value) Add(o Value) Value { return Value{scaled: m.scaled + o.scaled} }
func (m Value) Sub(o Value) Value { return Value{scaled: m.scaled - o.scaled} }
func (m Value) Neg() Value        { return Value{scaled: -m.scaled} }
func (m Value) Abs() Value {
	if m.scaled < 0 {
		return Value{scaled: -m.scaled}
	}
	return Value{scaled: m.scaled}
}
func (m Value) Copy() Value { return Value{scaled: m.scaled, raw: m.raw} }

// Mul multiplies two Value operands using a 128-bit intermediate. Truncates.
func (m Value) Mul(o Value) Value {
	neg := (m.scaled < 0) != (o.scaled < 0)
	a, b := abs64(m.scaled), abs64(o.scaled)
	hi, lo := bits.Mul64(a, b)
	q, _ := bits.Div64(hi, lo, Scale)
	r := int64(q)
	if neg {
		r = -r
	}
	return Value{scaled: r}
}

// MulInt multiplies by a plain integer. Use when multiplier has no decimal
// part — faster than Mul.
func (m Value) MulInt(n int64) Value { return Value{scaled: m.scaled * n} }

// Div divides truncating toward zero. Returns 0 if divisor is zero.
func (m Value) Div(o Value) Value {
	if o.scaled == 0 {
		return Value{}
	}
	neg := (m.scaled < 0) != (o.scaled < 0)
	a, b := abs64(m.scaled), abs64(o.scaled)
	hi, lo := bits.Mul64(a, Scale)
	q, _ := bits.Div64(hi, lo, b)
	r := int64(q)
	if neg {
		r = -r
	}
	return Value{scaled: r}
}

func (m Value) DivInt(n int64) Value {
	if n == 0 {
		return Value{}
	}
	return Value{scaled: m.scaled / n}
}

// Truncate reduces to given number of decimal places (0-6), toward zero.
func (m Value) Truncate(places int32) Value { return m.RoundDown(places) }

func (m Value) IsZero() bool           { return m.scaled == 0 }
func (m Value) IsPositive() bool       { return m.scaled > 0 }
func (m Value) IsPositiveOrZero() bool { return m.scaled >= 0 }
func (m Value) IsNegative() bool       { return m.scaled < 0 }
func (m Value) IsNegativeOrZero() bool { return m.scaled <= 0 }

func (m Value) Equal(o Value) bool              { return m.scaled == o.scaled }
func (m Value) GreaterThan(o Value) bool        { return m.scaled > o.scaled }
func (m Value) GreaterThanOrEqual(o Value) bool { return m.scaled >= o.scaled }
func (m Value) LessThan(o Value) bool            { return m.scaled < o.scaled }
func (m Value) LessThanOrEqual(o Value) bool     { return m.scaled <= o.scaled }

func (m Value) Cmp(o Value) int {
	switch {
	case m.scaled < o.scaled:
		return -1
	case m.scaled > o.scaled:
		return 1
	default:
		return 0
	}
}

func (m Value) Sign() int {
	switch {
	case m.scaled < 0:
		return -1
	case m.scaled > 0:
		return 1
	default:
		return 0
	}
}

func abs64(x int64) uint64 {
	if x < 0 {
		return uint64(-x)
	}
	return uint64(x)
}

// --- Rounding (places clamped to [0,6]) ---

func (m Value) Round(places int32) Value {
	div, ok := roundDiv(places)
	if !ok {
		return m
	}
	half := div / 2
	s := m.scaled
	if s >= 0 {
		return Value{scaled: ((s + half) / div) * div}
	}
	return Value{scaled: ((s - half) / div) * div}
}

// RoundBank uses banker's rounding (half-to-even).
func (m Value) RoundBank(places int32) Value {
	div, ok := roundDiv(places)
	if !ok {
		return m
	}
	half := div / 2
	s := m.scaled
	q := s / div
	r := s % div
	abs := r
	if abs < 0 {
		abs = -abs
	}
	switch {
	case abs < half:
		// round toward zero (already truncated)
	case abs > half:
		if s >= 0 {
			q++
		} else {
			q--
		}
	default:
		// exactly half — round to even
		if q%2 != 0 {
			if s >= 0 {
				q++
			} else {
				q--
			}
		}
	}
	return Value{scaled: q * div}
}

// RoundCash rounds to the nearest cash-interval (e.g., 5 → round to 0.05).
// Behavior matches shopspring/decimal.RoundCash.
func (m Value) RoundCash(interval uint8) Value {
	d := decimal.New(m.scaled, -6).RoundCash(interval)
	return Value{scaled: d.Shift(6).Truncate(0).IntPart(), raw: m.raw}
}

func (m Value) RoundCeil(places int32) Value {
	div, ok := roundDiv(places)
	if !ok {
		return m
	}
	q := m.scaled / div
	r := m.scaled % div
	if r > 0 {
		q++
	}
	return Value{scaled: q * div}
}

func (m Value) RoundFloor(places int32) Value {
	div, ok := roundDiv(places)
	if !ok {
		return m
	}
	q := m.scaled / div
	r := m.scaled % div
	if r < 0 {
		q--
	}
	return Value{scaled: q * div}
}

func (m Value) RoundDown(places int32) Value {
	div, ok := roundDiv(places)
	if !ok {
		return m
	}
	return Value{scaled: (m.scaled / div) * div}
}

func (m Value) RoundUp(places int32) Value {
	div, ok := roundDiv(places)
	if !ok {
		return m
	}
	q := m.scaled / div
	r := m.scaled % div
	switch {
	case r > 0:
		q++
	case r < 0:
		q--
	}
	return Value{scaled: q * div}
}

// roundDiv returns the divisor to drop digits below `places`.
// Returns (_, false) when places >= 6 (nothing to round).
func roundDiv(places int32) (int64, bool) {
	if places >= 6 {
		return 0, false
	}
	if places < 0 {
		places = 0
	}
	div := int64(1)
	for i := int32(0); i < 6-places; i++ {
		div *= 10
	}
	return div, true
}

// --- Decimal interop ---

func (m Value) Decimal() decimal.Decimal {
	return decimal.New(m.scaled, -6)
}

// --- String / formatting ---

// String matches shopspring/decimal: strips trailing zeros, no trailing dot.
func (m Value) String() string {
	s := m.scaled
	neg := s < 0
	if neg {
		s = -s
	}
	intPart := s / Scale
	fracPart := s % Scale

	sign := ""
	if neg {
		sign = "-"
	}
	if fracPart == 0 {
		return sign + strconv.FormatInt(intPart, 10)
	}
	frac := padLeft(strconv.FormatInt(fracPart, 10), 6, '0')
	// strip trailing zeros
	frac = strings.TrimRight(frac, "0")
	return sign + strconv.FormatInt(intPart, 10) + "." + frac
}

func padLeft(s string, width int, pad byte) string {
	if len(s) >= width {
		return s
	}
	buf := make([]byte, width)
	for i := 0; i < width-len(s); i++ {
		buf[i] = pad
	}
	copy(buf[width-len(s):], s)
	return string(buf)
}

func insertComma(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	offset := n % 3
	if offset > 0 {
		b.WriteString(s[:offset])
		if n > offset {
			b.WriteString(",")
		}
	}
	for i := offset; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteString(",")
		}
	}
	return b.String()
}

func formatNumber(value float64, precision int, useComma bool) string {
	format := fmt.Sprintf("%%.%df", precision)
	s := fmt.Sprintf(format, value)

	parts := strings.Split(s, ".")
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = parts[1]
	}
	if useComma {
		neg := strings.HasPrefix(intPart, "-")
		if neg {
			intPart = insertComma(intPart[1:])
			intPart = "-" + intPart
		} else {
			intPart = insertComma(intPart)
		}
	}
	if precision == 0 {
		return intPart
	}
	return intPart + "." + fracPart
}

func (m Value) FormatString() string {
	return formatNumber(m.Round(2).Float64(), 2, true)
}

func (m Value) FormatNumber() string {
	return formatNumber(m.Round(2).Float64(), 2, false)
}

func (m Value) FormatNumberWithPrecision(precision int) string {
	return formatNumber(m.Round(int32(precision)).Float64(), precision, false)
}

func (m Value) FormatNumberWithoutDecimal() string {
	return m.FormatNumberWithPrecision(0)
}

// --- JSON / XML / Text / YAML ---

func (m Value) MarshalJSON() ([]byte, error) { return []byte(m.String()), nil }

func (m *Value) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "null" || s == "" {
		*m = Value{}
		return nil
	}
	return m.setFromString(s)
}

func (m Value) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(m.String(), start)
}

func (m *Value) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	return m.setFromString(s)
}

func (m Value) MarshalText() ([]byte, error)     { return []byte(m.String()), nil }
func (m *Value) UnmarshalText(text []byte) error { return m.setFromString(string(text)) }

func (m Value) MarshalYAML() (interface{}, error) { return m.String(), nil }

func (m *Value) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}
	switch v := raw.(type) {
	case string:
		return m.setFromString(v)
	case int:
		*m = FromInt(v)
	case int64:
		*m = FromInt64(v)
	case float64:
		*m = FromFloat(v)
	default:
		return fmt.Errorf("unsupported YAML value for amountfast.Value: %T", raw)
	}
	return nil
}

// --- BSON ---

func (m Value) MarshalBSONValue() (bsontype.Type, []byte, error) {
	d128, _ := primitive.ParseDecimal128(m.String())
	return bson.TypeDecimal128, bsoncore.AppendDecimal128(nil, d128), nil
}

func (m *Value) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	switch t {
	case bson.TypeDecimal128:
		d128, _, ok := bsoncore.ReadDecimal128(data)
		if !ok {
			return errors.New("invalid decimal128")
		}
		return m.setFromString(d128.String())
	case bson.TypeInt32:
		i, _, ok := bsoncore.ReadInt32(data)
		if !ok {
			return errors.New("invalid int32")
		}
		*m = FromInt(int(i))
	case bson.TypeInt64:
		i, _, ok := bsoncore.ReadInt64(data)
		if !ok {
			return errors.New("invalid int64")
		}
		*m = FromInt64(i)
	case bson.TypeDouble:
		v, _, ok := bsoncore.ReadDouble(data)
		if !ok {
			return errors.New("invalid float64")
		}
		*m = FromFloat(v)
	default:
		return fmt.Errorf("unsupported BSON type: %s", t)
	}
	return nil
}

// --- SQL driver ---

func (m Value) Value() (driver.Value, error) { return m.String(), nil }

func (m *Value) Scan(value interface{}) error {
	switch v := value.(type) {
	case float64:
		*m = FromFloat(v)
	case int64:
		*m = FromInt64(v)
	case string:
		return m.setFromString(v)
	case []byte:
		return m.setFromString(string(v))
	default:
		return fmt.Errorf("cannot scan value of type %T into Value", v)
	}
	return nil
}

func (m *Value) setFromString(s string) error {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		*m = Value{}
		return nil
	}
	body := trimmed
	neg := false
	switch body[0] {
	case '-':
		neg = true
		body = body[1:]
	case '+':
		body = body[1:]
	}

	intStr, fracStr, _ := strings.Cut(body, ".")
	if intStr == "" {
		intStr = "0"
	}
	ip, err := strconv.ParseInt(intStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid Value %q: %w", s, err)
	}
	if len(fracStr) > 6 {
		fracStr = fracStr[:6]
	}
	for len(fracStr) < 6 {
		fracStr += "0"
	}
	fp, err := strconv.ParseInt(fracStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid fraction %q: %w", fracStr, err)
	}

	result := ip*Scale + fp
	if neg {
		result = -result
	}
	m.scaled = result
	m.raw = s
	return nil
}
