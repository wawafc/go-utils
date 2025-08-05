package amount

import (
	"database/sql/driver"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type Value struct {
	decimal decimal.Decimal
	raw     string
}

func FromString(s string) (Value, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Value{}, err
	}
	return Value{decimal: d, raw: s}, nil
}

func FromFloat(f float64) Value {
	d := decimal.NewFromFloat(f)
	return Value{decimal: d}
}

func FromInt(i int) Value {
	return Value{decimal: decimal.NewFromInt(int64(i))}
}

func FromInt64(i int64) Value {
	return Value{decimal: decimal.NewFromInt(i)}
}

func FromInt32(i int32) Value {
	return Value{decimal: decimal.NewFromInt32(i)}
}

func FromDecimal(d decimal.Decimal) Value {
	return Value{decimal: d}
}

func (m Value) MarshalJSON() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *Value) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), "\"")
	if s == "null" {
		m.decimal = decimal.NewFromFloat(0)
		return nil
	}

	value, err := decimal.NewFromString(s)
	if err != nil {
		return err
	}
	m.decimal = value
	m.raw = s
	return nil
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

func (m *Value) UnmarshalText(text []byte) error {
	return m.setFromString(string(text))
}

func (m Value) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *Value) setFromString(s string) error {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return err
	}
	m.decimal = d
	m.raw = s
	return nil
}

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
		m.decimal = decimal.NewFromInt32(i)
	case bson.TypeInt64:
		i, _, ok := bsoncore.ReadInt64(data)
		if !ok {
			return errors.New("invalid int64")
		}
		m.decimal = decimal.NewFromInt(i)
	case bson.TypeDouble:
		f, _, ok := bsoncore.ReadDouble(data)
		if !ok {
			return errors.New("invalid float64")
		}
		m.decimal = decimal.NewFromFloat(f)
	default:
		return fmt.Errorf("unsupported BSON type: %s", t)
	}
	return nil
}

func (m *Value) Scan(value interface{}) error {
	switch v := value.(type) {
	case float64:
		m.decimal = decimal.NewFromFloat(v)
	case int64:
		m.decimal = decimal.NewFromInt(v)
	case string:
		return m.setFromString(v)
	case []byte:
		return m.setFromString(string(v))
	default:
		return fmt.Errorf("cannot scan value of type %T into Value", v)
	}
	return nil
}

func (m Value) MarshalYAML() (interface{}, error) {
	return m.String(), nil
}

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
		return fmt.Errorf("unsupported YAML value for amount.Value: %T", raw)
	}
	return nil
}

func (m Value) Value() (driver.Value, error) {
	return m.decimal.String(), nil
}

func (m Value) Add(other Value) Value       { return Value{decimal: m.decimal.Add(other.decimal)} }
func (m Value) Sub(other Value) Value       { return Value{decimal: m.decimal.Sub(other.decimal)} }
func (m Value) Mul(other Value) Value       { return Value{decimal: m.decimal.Mul(other.decimal)} }
func (m Value) Div(other Value) Value       { return Value{decimal: m.decimal.Div(other.decimal)} }
func (m Value) Truncate(places int32) Value { return Value{decimal: m.decimal.Truncate(places)} }

func (m Value) String() string               { return m.decimal.String() }
func (m Value) Float64() float64             { f, _ := m.decimal.Float64(); return f }
func (m Value) Int64() int64                 { return m.decimal.IntPart() }
func (m Value) Int32() int32                 { return int32(m.decimal.IntPart()) }
func (m Value) Int() int                     { return int(m.decimal.IntPart()) }
func (m Value) IsZero() bool                 { return m.decimal.IsZero() }
func (m Value) Equal(other Value) bool       { return m.decimal.Equal(other.decimal) }
func (m Value) GreaterThan(other Value) bool { return m.decimal.GreaterThan(other.decimal) }
func (m Value) LessThan(other Value) bool    { return m.decimal.LessThan(other.decimal) }

func (m Value) Round(places int32) Value {
	return Value{decimal: m.decimal.Round(places)}
}

func (m Value) RoundBank(places int32) Value {
	return Value{decimal: m.decimal.RoundBank(places)}
}

func (m Value) RoundCash(interval uint8) Value {
	return Value{decimal: m.decimal.RoundCash(interval)}
}

func (m Value) RoundCeil(places int32) Value {
	return Value{decimal: m.decimal.RoundCeil(places)}
}

func (m Value) RoundDown(places int32) Value {
	return Value{decimal: m.decimal.RoundDown(places)}
}

func (m Value) RoundFloor(places int32) Value {
	return Value{decimal: m.decimal.RoundFloor(places)}
}

func (m Value) RoundUp(places int32) Value {
	return Value{decimal: m.decimal.RoundUp(places)}
}

func (m Value) Abs() Value {
	return Value{decimal: m.decimal.Abs()}
}

func (m Value) Neg() Value {
	return Value{decimal: m.decimal.Neg()}
}

func (m Value) Copy() Value {
	return Value{decimal: m.decimal.Copy()}
}

func (m Value) Raw() string {
	return m.raw
}

// insertComma adds commas to a numeric string (e.g., "1234567" → "1,234,567")
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

// formatNumber formats a float with optional commas and decimal precision.
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
		intPart = insertComma(intPart)
	}

	if precision == 0 {
		return intPart
	}
	return intPart + "." + fracPart
}

func (m Value) FormatString() string {
	f, _ := m.decimal.Round(2).Float64()
	return formatNumber(f, 2, true) // comma + dot
}

func (m Value) FormatNumber() string {
	f, _ := m.decimal.Round(2).Float64()
	return formatNumber(f, 2, false) // dot only
}

func (m Value) FormatNumberWithPrecision(precision int) string {
	f, _ := m.decimal.Round(int32(precision)).Float64()
	return formatNumber(f, precision, false)
}

func (m Value) FormatNumberWithoutDecimal() string {
	return m.FormatNumberWithPrecision(0)
}

func (m Value) Cmp(other Value) int {
	return m.decimal.Cmp(other.decimal)
}

func (m Value) Sign() int {
	return m.decimal.Sign()
}

func (m Value) Decimal() decimal.Decimal {
	return m.decimal
}
