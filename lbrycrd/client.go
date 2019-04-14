package lbrycrd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/lbryio/lbry.go/extras/errors"
	upstream "github.com/lbryio/lbry.go/lbrycrd"
	"github.com/mitchellh/mapstructure"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

var defaultClient *upstream.Client

// LBRYcrdURL is the connection string for lbrycrd and is set from the configuration
var LBRYcrdURL string

// Init initializes a client with settings from the configuration of chainquery
func Init() *upstream.Client {
	lbrycrdClient, err := upstream.New(LBRYcrdURL)
	if err != nil {
		logrus.Panic("Initializing LBRYcrd Client: ", err)
	}
	defaultClient = lbrycrdClient
	_, err = GetBalance()
	if err != nil {
		logrus.Panicf("Error connecting to lbrycrd: %+v", err)
	}
	return lbrycrdClient
}

func call(response interface{}, command string, params ...interface{}) error {
	result, err := callNoDecode(command, params...)
	if err != nil {
		return err
	}
	return decode(result, response)
}

func callNoDecode(command string, params ...interface{}) (interface{}, error) {
	//logrus.Debug("jsonrpc: " + command + " " + debugParams(params))
	var err error

	encodedParams := make([]json.RawMessage, len(params))
	for i, p := range params {
		encodedParams[i], err = json.Marshal(p)
		if err != nil {
			return nil, errors.Err(err)
		}
	}

	encodedRes, err := defaultClient.RawRequest(command, encodedParams)
	if err != nil {
		return nil, err
	}

	var res interface{}
	decoder := json.NewDecoder(bytes.NewReader(encodedRes))
	decoder.UseNumber()
	err = decoder.Decode(&res)
	return res, errors.Err(err)

}

func decode(data interface{}, targetStruct interface{}) error {
	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   targetStruct,
		TagName:  "json",
		//WeaklyTypedInput: true,
		DecodeHook: fixDecodeProto,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	err = decoder.Decode(data)
	if err != nil {
		return errors.Wrap(err, 0)
	}
	return nil
}

//ToDo - Can we just use the decodeInt function? Is it double for nothing?
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
		return decimal.Decimal{}, errors.Base("unexpected number type ")
	}

	dec, err := decimal.NewFromString(number)
	if err != nil {
		return decimal.Decimal{}, errors.Wrap(err, 0)
	}

	return dec, nil
}

type protoFunc func(interface{}) (interface{}, error)

type protoMap map[reflect.Type]protoFunc

var decodeMap = protoMap{
	reflect.TypeOf(uint64(0)):         decodeInt,
	reflect.TypeOf([]byte{}):          decodeBytes,
	reflect.TypeOf(decimal.Decimal{}): decodeFloat,
}

func fixDecodeProto(src, dest reflect.Type, data interface{}) (interface{}, error) {

	if f, ok := decodeMap[dest]; ok {
		return f(data)
	}

	return data, nil
}

func decodeInt(data interface{}) (interface{}, error) {
	if n, ok := data.(json.Number); ok {
		val, err := n.Int64()
		if err != nil {
			return nil, errors.Wrap(err, 0)
		} else if val < 0 {
			return nil, errors.Base("must be unsigned int")
		}
		return uint64(val), nil
	}
	return data, nil
}

func decodeFloat(data interface{}) (interface{}, error) {
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
	return data, nil
}

func decodeBytes(data interface{}) (interface{}, error) {
	if s, ok := data.(string); ok {
		return []byte(s), nil
	}
	return data, nil
}
