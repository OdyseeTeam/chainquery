package lbrycrd

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-errors/errors"
	lbryschema "github.com/lbryio/lbryschema.go/pb"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

func (d *Client) callNoDecode(command string, params ...interface{}) (interface{}, error) {
	//logrus.Debugln("jsonrpc: " + command + " " + debugParams(params))
	if len(params) == 0 {
		r, err := d.conn.Call(command)
		if err != nil {
			return nil, err
		}
		return r.Result, nil
	}
	if len(params) == 1 {
		r, err := d.conn.Call(command, params[0])
		if err != nil {
			return nil, err
		}
		return r.Result, nil
	}
	if len(params) == 2 {
		r, err := d.conn.Call(command, params[0], params[1])
		if err != nil {
			return nil, err
		}
		return r.Result, nil
	}
	logrus.Error("parameter size is greater than 1")
	return nil, nil
}

func decodeNumber(data interface{}) (decimal.Decimal, error) {
	var number string
	switch d := data.(type) {
	case json.Number:
		number = d.String()
	case string:
		number = d
	case nil:
		number = "0"
	default:
		fmt.Printf("I don't know about type %T!\n", d)
		return decimal.Decimal{}, errors.New("unexpected number type ")
	}

	dec, err := decimal.NewFromString(number)
	if err != nil {
		return decimal.Decimal{}, errors.Wrap(err, 0)
	}

	return dec, nil
}

func FixDecodeProto(src, dest reflect.Type, data interface{}) (interface{}, error) {
	switch dest {
	case reflect.TypeOf(uint64(0)):
		if n, ok := data.(json.Number); ok {
			val, err := n.Int64()
			if err != nil {
				return nil, errors.Wrap(err, 0)
			} else if val < 0 {
				return nil, errors.New("must be unsigned int")
			}
			return uint64(val), nil
		}
	case reflect.TypeOf([]byte{}):
		if s, ok := data.(string); ok {
			return []byte(s), nil
		}

	case reflect.TypeOf(decimal.Decimal{}):
		if n, ok := data.(json.Number); ok {
			val, err := n.Float64()
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
			return decimal.NewFromFloat(val), nil
		} else if s, ok := data.(string); ok {
			d, err := decimal.NewFromString(s)
			if err != nil {
				return nil, errors.Wrap(err, 0)
			}
			return d, nil
		}

	case reflect.TypeOf(lbryschema.Metadata_Version(0)):
		val, err := GetEnumVal(lbryschema.Metadata_Version_value, data)
		return lbryschema.Metadata_Version(val), err
	case reflect.TypeOf(lbryschema.Metadata_Language(0)):
		val, err := GetEnumVal(lbryschema.Metadata_Language_value, data)
		return lbryschema.Metadata_Language(val), err

	case reflect.TypeOf(lbryschema.Stream_Version(0)):
		val, err := GetEnumVal(lbryschema.Stream_Version_value, data)
		return lbryschema.Stream_Version(val), err

	case reflect.TypeOf(lbryschema.Claim_Version(0)):
		val, err := GetEnumVal(lbryschema.Claim_Version_value, data)
		return lbryschema.Claim_Version(val), err
	case reflect.TypeOf(lbryschema.Claim_ClaimType(0)):
		val, err := GetEnumVal(lbryschema.Claim_ClaimType_value, data)
		return lbryschema.Claim_ClaimType(val), err

	case reflect.TypeOf(lbryschema.Fee_Version(0)):
		val, err := GetEnumVal(lbryschema.Fee_Version_value, data)
		return lbryschema.Fee_Version(val), err
	case reflect.TypeOf(lbryschema.Fee_Currency(0)):
		val, err := GetEnumVal(lbryschema.Fee_Currency_value, data)
		return lbryschema.Fee_Currency(val), err

	case reflect.TypeOf(lbryschema.Source_Version(0)):
		val, err := GetEnumVal(lbryschema.Source_Version_value, data)
		return lbryschema.Source_Version(val), err
	case reflect.TypeOf(lbryschema.Source_SourceTypes(0)):
		val, err := GetEnumVal(lbryschema.Source_SourceTypes_value, data)
		return lbryschema.Source_SourceTypes(val), err

	case reflect.TypeOf(lbryschema.KeyType(0)):
		val, err := GetEnumVal(lbryschema.KeyType_value, data)
		return lbryschema.KeyType(val), err

	case reflect.TypeOf(lbryschema.Signature_Version(0)):
		val, err := GetEnumVal(lbryschema.Signature_Version_value, data)
		return lbryschema.Signature_Version(val), err

	case reflect.TypeOf(lbryschema.Certificate_Version(0)):
		val, err := GetEnumVal(lbryschema.Certificate_Version_value, data)
		return lbryschema.Certificate_Version(val), err
	}

	return data, nil
}

func GetEnumVal(enum map[string]int32, data interface{}) (int32, error) {
	s, ok := data.(string)
	if !ok {
		return 0, errors.New("expected a string")
	}
	val, ok := enum[s]
	if !ok {
		return 0, errors.New("invalid enum key")
	}
	return val, nil
}
