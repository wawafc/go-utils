package money

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/leekchan/accounting"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
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

func (m Money) MarshalBSONValue() (bsontype.Type, []byte, error) {
	s := m.d.String()
	d, _ := primitive.ParseDecimal128(s)
	return bsontype.Decimal128, bsoncore.AppendDecimal128([]byte{}, d), nil
}

func (m *Money) UnmarshalBSONValue(dataType bsontype.Type, data []byte) error {
	switch dataType {
	case bsontype.Decimal128:
		value, _, ok := bsoncore.ReadDecimal128(data)
		if !ok {
			return errors.New("Can't convert to Decimal128 " + string(data))
		}
		d, err := decimal.NewFromString(value.String())
		if err != nil {
			return err
		}
		m.d = d
	case bsontype.Int32:
		i, _, ok := bsoncore.ReadInt32(data)
		if !ok {
			return errors.New("Can't convert to Decimal128 " + string(data))
		}
		m.d = decimal.NewFromInt32(i)
	case bsontype.Int64:
		i, _, ok := bsoncore.ReadInt64(data)
		if !ok {
			return errors.New("Can't convert to Decimal128 " + string(data))
		}
		m.d = decimal.NewFromInt(i)
	case bsontype.Double:
		i, _, ok := bsoncore.ReadDouble(data)
		if !ok {
			return errors.New("Can't convert to Decimal128 " + string(data))
		}
		m.d = decimal.NewFromFloat(i)
	default:
		return errors.New("Can't unmarshal BSON value as data type " + dataType.String() + " data is " + string(data))
	}
	return nil
}

func (m Money) MarshalJSON() ([]byte, error) {
	f, _ := m.d.Float64()
	return json.Marshal(f)
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
