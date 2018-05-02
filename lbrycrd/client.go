package lbrycrd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/lbryio/lbry.go/errors"
	upstream "github.com/lbryio/lbry.go/lbrycrd"
	lbryschema "github.com/lbryio/lbryschema.go/pb"

	"github.com/mitchellh/mapstructure"
	"github.com/shopspring/decimal"
)

var defaultClient *upstream.Client

// LBRYcrdURL is the connection string for lbrycrd and is set from the configuration
var LBRYcrdURL string

// Init initializes a client with settings from the configuration of chainquery
func Init() *upstream.Client {
	lbrycrdClient, err := upstream.New(LBRYcrdURL)
	if err != nil {
		panic(err)
	}
	defaultClient = lbrycrdClient
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
	reflect.TypeOf(uint64(0)):                         decodeInt,
	reflect.TypeOf([]byte{}):                          decodeBytes,
	reflect.TypeOf(decimal.Decimal{}):                 decodeFloat,
	reflect.TypeOf(lbryschema.Metadata_Version(0)):    decodeMetaDataVersion,
	reflect.TypeOf(lbryschema.Metadata_Language(0)):   decodeMetaDataLanguage,
	reflect.TypeOf(lbryschema.Stream_Version(0)):      decodeStreamVersion,
	reflect.TypeOf(lbryschema.Claim_Version(0)):       decodeClaimVersion,
	reflect.TypeOf(lbryschema.Claim_ClaimType(0)):     decodeClaimType,
	reflect.TypeOf(lbryschema.Fee_Version(0)):         decodeFeeVersion,
	reflect.TypeOf(lbryschema.Fee_Currency(0)):        decodeFeeCurrency,
	reflect.TypeOf(lbryschema.Source_Version(0)):      decodeSourceVersion,
	reflect.TypeOf(lbryschema.Source_SourceTypes(0)):  decodeSourceTypes,
	reflect.TypeOf(lbryschema.KeyType(0)):             decodeKeyType,
	reflect.TypeOf(lbryschema.Signature_Version(0)):   decodeSignatureVersion,
	reflect.TypeOf(lbryschema.Certificate_Version(0)): decodeCertificateVersion,
}

func fixDecodeProto(src, dest reflect.Type, data interface{}) (interface{}, error) {

	if f, ok := decodeMap[dest]; ok {
		return f(data)
	}

	return data, nil
}

func getEnumVal(enum map[string]int32, data interface{}) (int32, error) {
	s, ok := data.(string)
	if !ok {
		return 0, errors.Base("expected a string")
	}
	val, ok := enum[s]
	if !ok {
		return 0, errors.Base("invalid enum key")
	}
	return val, nil
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

func decodeMetaDataVersion(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Metadata_Version_value, data)
	return lbryschema.Metadata_Version(val), err
}

func decodeMetaDataLanguage(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Metadata_Language_value, data)
	return lbryschema.Metadata_Language(val), err
}

func decodeStreamVersion(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Stream_Version_value, data)
	return lbryschema.Stream_Version(val), err
}

func decodeClaimVersion(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Claim_Version_value, data)
	return lbryschema.Claim_Version(val), err
}

func decodeClaimType(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Claim_ClaimType_value, data)
	return lbryschema.Claim_ClaimType(val), err
}

func decodeFeeVersion(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Fee_Version_value, data)
	return lbryschema.Fee_Version(val), err
}

func decodeFeeCurrency(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Fee_Currency_value, data)
	return lbryschema.Fee_Currency(val), err
}

func decodeSourceVersion(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Source_Version_value, data)
	return lbryschema.Source_Version(val), err
}

func decodeSourceTypes(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Source_SourceTypes_value, data)
	return lbryschema.Source_SourceTypes(val), err
}

func decodeKeyType(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.KeyType_value, data)
	return lbryschema.KeyType(val), err
}

func decodeSignatureVersion(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Signature_Version_value, data)
	return lbryschema.Signature_Version(val), err
}

func decodeCertificateVersion(data interface{}) (interface{}, error) {
	val, err := getEnumVal(lbryschema.Certificate_Version_value, data)
	return lbryschema.Certificate_Version(val), err
}
