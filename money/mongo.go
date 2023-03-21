package money

import (
	"errors"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/x/bsonx/bsoncore"
)

//func (m Money) MarshalBSON() ([]byte, error) {
//	s := m.d.String()
//	d, err := primitive.ParseDecimal128(s)
//	if err != nil {
//		return nil, err
//	}
//	return bsoncore.AppendDecimal128([]byte{}, d), nil
//}
//
//func (m Money) UnmarshalBSON(data []byte) error {
//	return nil
//}

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

func (m Money) Decimal128() primitive.Decimal128 {
	s := m.d.String()
	d, _ := primitive.ParseDecimal128(s)
	return d
}
