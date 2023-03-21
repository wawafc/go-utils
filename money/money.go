package money

import (
	"database/sql/driver"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math/big"

	"github.com/leekchan/accounting"
	"github.com/shopspring/decimal"
)

func NewMoneyFromString(value string) (Money, error) {
	rtn, err := decimal.NewFromString(value)
	if err != nil {
		return Money{}, err
	}
	return Money{rtn}, nil
}

func NewMoneyFromFloat(value float64) Money {
	return Money{decimal.NewFromFloat(value)}
}

func NewMoneyFromDecimal(d decimal.Decimal) Money {
	return Money{d}
}

// Money handles monetary data, that used decimal to do operations on code,
// and used decimal128 to store in mongo database
// https://docs.mongodb.com/manual/tutorial/model-monetary-data/#numeric-decimal

type Money struct {
	d decimal.Decimal
}

// Scan implements the sql.Scanner interface for database deserialization.
func (d *Money) Scan(value interface{}) error {
	// first try to see if the data is stored in database as a Numeric datatype
	switch v := value.(type) {

	case float32:
		*d = NewMoneyFromFloat(float64(v))
		return nil

	case float64:
		// numeric in sqlite3 sends us float64
		*d = NewMoneyFromFloat(v)
		return nil

	case int64:
		// at least in sqlite3 when the value is 0 in db, the data is sent
		// to us as an int64 instead of a float64 ...
		*d = NewMoneyFromFloat(float64(v))
		return nil

	case string:
		m, err := NewMoneyFromString(v)
		if err != nil {
			return err
		}
		*d = m
		return nil

	default:
		// default is trying to interpret value stored as string
		return errors.New("unsupported type")
	}
}

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.d.String()), nil
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var a float64
	if err := json.Unmarshal(data, &a); err == nil {
		m.d = decimal.NewFromFloat(a)
		return nil
	}

	var b string
	if err := json.Unmarshal(data, &b); err == nil {
		m.d, err = decimal.NewFromString(b)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("cannot unmarshal with other types")

}

func (m Money) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(m.Round(2).String(), start)
}
func (m *Money) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var v string
	err := d.DecodeElement(&v, &start)
	if err != nil {
		return err
	}
	value, err := decimal.NewFromString(v)
	if err != nil {
		return err
	}
	m.d = value
	return nil
}

func (m *Money) UnmarshalXMLAttr(attr xml.Attr) error {
	value, err := decimal.NewFromString(attr.Value)
	if err != nil {
		return err
	}
	m.d = value
	return nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for XML
// deserialization.
func (m *Money) UnmarshalText(text []byte) error {
	str := string(text)

	dec, err := decimal.NewFromString(str)
	m.d = dec
	if err != nil {
		return fmt.Errorf("error decoding string '%s': %s", str, err)
	}

	return nil
}

// MarshalText implements the encoding.TextMarshaler interface for XML
// serialization.
func (m Money) MarshalText() (text []byte, err error) {
	return []byte(m.String()), nil
}

func (m Money) Div(value Money) Money {
	return Money{m.d.Div(value.d)}
}

func (m Money) FormatString() string {
	f, _ := m.d.Round(2).Float64()
	return accounting.FormatNumberFloat64(f, 2, ",", ".")
}

func (m Money) FormatNumber() string {
	f, _ := m.d.Round(2).Float64()
	return accounting.FormatNumberFloat64(f, 2, "", ".")
}

func (m Money) FormatNumberWithPrecision(precision int) string {
	f, _ := m.d.Float64()
	return accounting.FormatNumberFloat64(f, precision, "", ".")
}

func (m Money) FormatNumberWithoutDecimal() string {
	f, _ := m.d.Round(0).Float64()
	return accounting.FormatNumberFloat64(f, 0, ",", ".")
}

func (m Money) FloorWithDecimal(index int32) Money {
	if index <= 0 {
		return m
	}
	return Money{m.d.Shift(index).Floor().Shift(-index)}
}

func (m Money) Sub(value Money) Money {
	return Money{m.d.Sub(value.d)}
}

func (m Money) Add(value Money) Money {
	return Money{m.d.Add(value.d)}
}

func (m Money) Shift(shift int32) Money {
	return Money{m.d.Shift(shift)}
}

func (m Money) Mul(value Money) Money {
	return Money{m.d.Mul(value.d)}
}

func (m Money) Round(num int32) Money {
	return Money{m.d.Round(num)}
}

func (m Money) RoundBank(num int32) Money {
	return Money{m.d.RoundBank(num)}
}

func (m Money) RoundCash(num uint8) Money {
	return Money{m.d.RoundCash(num)}
}

func (m Money) RoundCeil(num int32) Money {
	return Money{m.d.RoundCeil(num)}
}

func (m Money) RoundDown(num int32) Money {
	return Money{m.d.RoundDown(num)}
}

func (m Money) RoundFloor(num int32) Money {
	return Money{m.d.RoundFloor(num)}
}

func (m Money) RoundUp(num int32) Money {
	return Money{m.d.RoundUp(num)}
}

func (m Money) Abs() Money {
	return Money{m.d.Abs()}
}

func (m Money) Ceil() Money {
	return Money{m.d.Ceil()}
}

func (m Money) Cmp(value Money) int {
	return m.d.Cmp(value.d)
}

func (m Money) Equal(value Money) bool {
	return m.d.Equal(value.d)
}

func (m Money) LessThan(value Money) bool {
	return m.d.LessThan(value.d)
}

func (m Money) GreaterThan(value Money) bool {
	return m.d.GreaterThan(value.d)
}

func (m Money) LessThanOrEqual(value Money) bool {
	return m.d.LessThanOrEqual(value.d)
}

func (m Money) GreaterThanOrEqual(value Money) bool {
	return m.d.GreaterThanOrEqual(value.d)
}

func (m Money) Float64() float64 {
	v, _ := m.d.Float64()
	return v
}

func (m Money) Floor() Money {
	return Money{m.d.Floor()}
}

func (m Money) String() string {
	return m.d.String()
}

func (m Money) Neg() Money {
	return Money{m.d.Neg()}
}

func (m Money) Mod(value Money) Money {
	return Money{d: m.d.Mod(value.d)}
}

func (m Money) Pow(value Money) Money {
	return Money{d: m.d.Pow(value.d)}
}

func (m Money) IsInteger() bool {
	return m.d.IsInteger()
}

func (m Money) IsNegative() bool {
	return m.d.IsNegative()
}

func (m Money) IsPositive() bool {
	return m.d.IsPositive()
}

func (m Money) IsZero() bool {
	return m.d.IsZero()
}

func (m Money) Copy() Money {
	return Money{m.d.Copy()}
}
func (m Money) QuoRem(d2 Money, precision int32) (Money, Money) {
	dq, dr := m.d.QuoRem(d2.d, precision)
	return Money{dq}, Money{dr}
}
func (m Money) DivRound(d2 Money, precision int32) Money {
	return Money{m.d.DivRound(d2.d, precision)}
}
func (m Money) ExpHullAbrham(overallPrecision uint32) (Money, error) {
	d, err := m.d.ExpHullAbrham(overallPrecision)
	return Money{d}, err
}
func (m Money) ExpTaylor(precision int32) (Money, error) {
	d, err := m.d.ExpTaylor(precision)
	return Money{d}, err
}
func (m Money) NumDigits() int {
	return m.d.NumDigits()
}
func (m Money) Equals(d2 Money) bool {
	return m.d.Equals(d2.d)
}
func (m Money) Sign() int {
	return m.d.Sign()
}

func (m Money) Exponent() int32 {
	return m.d.Exponent()
}
func (m Money) Coefficient() *big.Int {
	return m.d.Coefficient()
}
func (m Money) CoefficientInt64() int64 {
	return m.d.CoefficientInt64()
}
func (m Money) IntPart() int64 {
	return m.d.IntPart()
}
func (m Money) BigInt() *big.Int {
	return m.d.BigInt()
}
func (m Money) BigFloat() *big.Float {
	return m.d.BigFloat()
}
func (m Money) Rat() *big.Rat {
	return m.d.Rat()
}
func (m Money) InexactFloat64() float64 {
	return m.d.InexactFloat64()
}
func (m Money) StringFixed(places int32) string {
	return m.d.StringFixed(places)
}
func (m Money) StringFixedBank(places int32) string {
	return m.d.StringFixedBank(places)
}
func (m Money) StringFixedCash(interval uint8) string {
	return m.d.StringFixedCash(interval)
}
func (m Money) Truncate(precision int32) Money {
	return Money{m.d.Truncate(precision)}
}
func (m Money) MarshalBinary() (data []byte, err error) {
	return m.d.MarshalBinary()
}
func (m Money) Value() (driver.Value, error) {
	return m.d.Value()
}
func (m Money) GobEncode() ([]byte, error) {
	return m.d.GobEncode()
}
func (m Money) StringScaled(exp int32) string {
	return m.d.StringScaled(exp)
}
func (m Money) Atan() Money {
	return Money{m.d.Atan()}
}
func (m Money) Sin() Money {
	return Money{m.d.Sin()}
}
func (m Money) Cos() Money {
	return Money{m.d.Cos()}
}
func (m Money) Tan() Money {
	return Money{m.d.Tan()}
}
