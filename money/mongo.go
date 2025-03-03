package money

import (
	"errors"
	"github.com/shopspring/decimal"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/x/bsonx/bsoncore"
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

func (m Money) MarshalBSONValue() (byte, []byte, error) {
	s := m.d.String()
	d, _ := bson.ParseDecimal128(s)
	h, l := d.GetBytes()
	return byte(bson.TypeDecimal128), bsoncore.AppendDecimal128([]byte{}, h, l), nil
}

func (m *Money) UnmarshalBSONValue(dataType bson.Type, data []byte) error {
	switch dataType {
	case bson.TypeDecimal128:
		h, l, _, ok := bsoncore.ReadDecimal128(data)
		if !ok {
			return errors.New("Can't convert to Decimal128 " + string(data))
		}
		d128 := bson.NewDecimal128(h, l)

		d, err := decimal.NewFromString(d128.String())
		if err != nil {
			return err
		}
		m.d = d
	case bson.TypeInt32:
		i, _, ok := bsoncore.ReadInt32(data)
		if !ok {
			return errors.New("Can't convert to Decimal128 " + string(data))
		}
		m.d = decimal.NewFromInt32(i)
	case bson.TypeInt64:
		i, _, ok := bsoncore.ReadInt64(data)
		if !ok {
			return errors.New("Can't convert to Decimal128 " + string(data))
		}
		m.d = decimal.NewFromInt(i)
	case bson.TypeDouble:
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

func (m Money) Decimal128() bson.Decimal128 {
	s := m.d.String()
	d, _ := bson.ParseDecimal128(s)
	return d
}
