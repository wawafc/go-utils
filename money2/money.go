package money2

import (
	"database/sql/driver"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

type Money struct {
	decimal decimal.Decimal
	raw     string
}

func FromString(s string) (Money, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Money{}, err
	}
	return Money{decimal: d, raw: s}, nil
}

func FromFloat(f float64) Money {
	d := decimal.NewFromFloat(f)
	return Money{decimal: d}
}

func FromDecimal(d decimal.Decimal) Money {
	return Money{decimal: d}
}

func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *Money) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	return m.setFromString(s)
}

func (m Money) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	return e.EncodeElement(m.String(), start)
}

func (m *Money) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	return m.setFromString(s)
}

func (m *Money) UnmarshalText(text []byte) error {
	return m.setFromString(string(text))
}

func (m Money) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *Money) setFromString(s string) error {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return err
	}
	m.decimal = d
	m.raw = s
	return nil
}

func (m Money) MarshalBSONValue() (bsontype.Type, []byte, error) {
	d128, _ := primitive.ParseDecimal128(m.String())
	return bson.TypeDecimal128, bsoncore.AppendDecimal128(nil, d128), nil
}

func (m *Money) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
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

func (m *Money) Scan(value interface{}) error {
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
		return fmt.Errorf("cannot scan value of type %T into Money", v)
	}
	return nil
}

func (m Money) Value() (driver.Value, error) {
	return m.decimal.String(), nil
}

func (m Money) Add(other Money) Money       { return Money{decimal: m.decimal.Add(other.decimal)} }
func (m Money) Sub(other Money) Money       { return Money{decimal: m.decimal.Sub(other.decimal)} }
func (m Money) Mul(other Money) Money       { return Money{decimal: m.decimal.Mul(other.decimal)} }
func (m Money) Div(other Money) Money       { return Money{decimal: m.decimal.Div(other.decimal)} }
func (m Money) Truncate(places int32) Money { return Money{decimal: m.decimal.Truncate(places)} }

func (m Money) String() string               { return m.decimal.String() }
func (m Money) Float64() float64             { f, _ := m.decimal.Float64(); return f }
func (m Money) IsZero() bool                 { return m.decimal.IsZero() }
func (m Money) Equal(other Money) bool       { return m.decimal.Equal(other.decimal) }
func (m Money) GreaterThan(other Money) bool { return m.decimal.GreaterThan(other.decimal) }
func (m Money) LessThan(other Money) bool    { return m.decimal.LessThan(other.decimal) }

func (m Money) Round(places int32) Money {
	return Money{decimal: m.decimal.Round(places)}
}

func (m Money) RoundBank(places int32) Money {
	return Money{decimal: m.decimal.RoundBank(places)}
}

func (m Money) RoundCash(interval uint8) Money {
	return Money{decimal: m.decimal.RoundCash(interval)}
}

func (m Money) RoundCeil(places int32) Money {
	return Money{decimal: m.decimal.RoundCeil(places)}
}

func (m Money) RoundDown(places int32) Money {
	return Money{decimal: m.decimal.RoundDown(places)}
}

func (m Money) RoundFloor(places int32) Money {
	return Money{decimal: m.decimal.RoundFloor(places)}
}

func (m Money) RoundUp(places int32) Money {
	return Money{decimal: m.decimal.RoundUp(places)}
}

func (m Money) Abs() Money {
	return Money{decimal: m.decimal.Abs()}
}

func (m Money) Neg() Money {
	return Money{decimal: m.decimal.Neg()}
}

func (m Money) Copy() Money {
	return Money{decimal: m.decimal.Copy()}
}

func (m Money) Raw() string {
	return m.raw
}
