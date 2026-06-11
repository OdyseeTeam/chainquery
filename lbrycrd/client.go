package lbrycrd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sync/atomic"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/lbryio/chainquery/metrics"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/mitchellh/mapstructure"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// LBRYcrdURL is the connection string for lbrycrd and is set from the configuration
var LBRYcrdURL string

// DefaultClientTimeout is the timeout applied to lbrycrd HTTP POST RPC calls.
var DefaultClientTimeout time.Duration

var rpcID uint64

const rpcTransportDeadlineDivisor = 2

type rpcRequest struct {
	Jsonrpc string            `json:"jsonrpc"`
	Method  string            `json:"method"`
	Params  []json.RawMessage `json:"params"`
	ID      uint64            `json:"id"`
}

type rpcResponse struct {
	Result json.RawMessage   `json:"result"`
	Error  *btcjson.RPCError `json:"error"`
}

// Init initializes a client with settings from the configuration of chainquery
func Init() {
	_, err := GetChainParams()
	if err != nil {
		panic(err)
	}
	_, err = GetBlockCount()
	if err != nil {
		logrus.Panicf("Error connecting to lbrycrd: %+v", err)
	}
}

func call(response interface{}, command string, params ...interface{}) error {
	result, err := callNoDecode(command, params...)
	if err != nil {
		return err
	}
	return decode(result, response)
}

func callNoDecode(command string, params ...interface{}) (result interface{}, err error) {
	//logrus.Debug("jsonrpc: " + command + " " + debugParams(params))
	start := time.Now()
	metrics.LBRYcrdRPCInflight.WithLabelValues(command).Inc()
	defer func() {
		metrics.LBRYcrdRPCInflight.WithLabelValues(command).Dec()
		resultLabel := "success"
		if err != nil {
			resultLabel = "error"
		}
		metrics.LBRYcrdRPCLatency.WithLabelValues(command, resultLabel).Observe(time.Since(start).Seconds())
	}()

	encodedParams := make([]json.RawMessage, len(params))
	for i, p := range params {
		encodedParams[i], err = json.Marshal(p)
		if err != nil {
			return nil, errors.Err(err)
		}
	}

	encodedRes, err := rawRequest(command, encodedParams)
	if err != nil {
		return nil, err
	}

	var res interface{}
	decoder := json.NewDecoder(bytes.NewReader(encodedRes))
	decoder.UseNumber()
	err = decoder.Decode(&res)
	return res, errors.Err(err)

}

func rawRequest(command string, params []json.RawMessage) ([]byte, error) {
	if command == "" {
		return nil, errors.Err("no method")
	}
	if params == nil {
		params = []json.RawMessage{}
	}
	endpoint, username, password, err := rpcEndpoint()
	if err != nil {
		return nil, errors.Err(err)
	}
	payload := rpcRequest{
		Jsonrpc: "1.0",
		ID:      atomic.AddUint64(&rpcID, 1),
		Method:  command,
		Params:  params,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, errors.Err(err)
	}
	ctx := context.Background()
	if DefaultClientTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultClientTimeout)
		defer cancel()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Err(err)
	}
	request.Close = true
	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth(username, password)
	client := newRPCHTTPClient(DefaultClientTimeout)
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.Err(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, errors.Err(err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, errors.Base("status code: %d, response: %q", response.StatusCode, string(responseBody))
	}
	var rpcResp rpcResponse
	err = json.Unmarshal(responseBody, &rpcResp)
	if err != nil {
		return nil, errors.Base("status code: %d, response: %q", response.StatusCode, string(responseBody))
	}
	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}
	return rpcResp.Result, nil
}

func newRPCHTTPClient(timeout time.Duration) http.Client {
	transportTimeout := rpcTransportTimeout(timeout)
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: transportTimeout, KeepAlive: transportTimeout}).DialContext,
		TLSHandshakeTimeout:   transportTimeout,
		ResponseHeaderTimeout: transportTimeout,
		IdleConnTimeout:       timeout,
		ExpectContinueTimeout: transportTimeout,
	}
	return http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func rpcTransportTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 0
	}
	if timeout < time.Second {
		return timeout
	}
	return timeout / rpcTransportDeadlineDivisor
}

func rpcEndpoint() (string, string, string, error) {
	parsedURL, err := url.Parse(LBRYcrdURL)
	if err != nil {
		return "", "", "", errors.Err(err)
	}
	if parsedURL.User == nil {
		return "", "", "", errors.Err("no userinfo")
	}
	password, _ := parsedURL.User.Password()
	scheme := parsedURL.Scheme
	switch scheme {
	case "rpc":
		scheme = "http"
	case "http", "https":
	default:
		return "", "", "", errors.Err("unsupported lbrycrd URL scheme %q", parsedURL.Scheme)
	}
	if parsedURL.Host == "" {
		return "", "", "", errors.Err("missing lbrycrd host")
	}
	endpoint := url.URL{
		Scheme:   scheme,
		Host:     parsedURL.Host,
		Path:     parsedURL.Path,
		RawQuery: parsedURL.RawQuery,
	}
	return endpoint.String(), parsedURL.User.Username(), password, nil
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

// ToDo - Can we just use the decodeInt function? Is it double for nothing?
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
