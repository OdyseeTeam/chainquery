package lbrycrd

import (
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/lbryio/lbry.go/errors"

	"github.com/mitchellh/mapstructure"
	"github.com/ybbus/jsonrpc"
)

// Client connects to a lbrycrd instance
type Client struct {
	conn *jsonrpc.RPCClient
}

var defaultClient *Client

func DefaultClient() *Client {
	if defaultClient == nil {
		panic("no default lbrycrd cilent")
	}
	return defaultClient
}

func SetDefaultClient(client *Client) {
	defaultClient = client
}

// New initializes a new instance of a Client. If the url cannot be parsed
// an error will be thrown. If the user information cannot be parsed, an
// error will be thrown.
func New(lbrycrdURL string) (*Client, error) {
	// Connect to local bitcoin core RPC server using HTTP POST mode.
	u, err := url.Parse(lbrycrdURL)
	if err != nil {
		return nil, errors.Err(err)
	}
	if u.User == nil {
		return nil, errors.Err("no userinfo")
	}

	password, _ := u.User.Password()
	url := "http://" + u.Host
	client := jsonrpc.NewRPCClient(url)
	client.SetBasicAuth(u.User.Username(), password)

	return &Client{client}, nil
}

//Performs a shutdown of the jsonrpc client.
func (client *Client) Shutdown() {
	client.Shutdown()
}

var errInsufficientFunds = errors.Base("Our wallet is running low. We've been notified, and we will refill it ASAP. Please try again in a little while, or email us at hello@lbry.io for more info.")

// response is the raw bytes of a JSON-RPC result, or the error if the response
// error object was non-null.
type response struct {
	result []byte
	err    error
}

func decode(data interface{}, targetStruct interface{}) error {
	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		Result:   targetStruct,
		TagName:  "json",
		//WeaklyTypedInput: true,
		DecodeHook: FixDecodeProto,
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

func debugParams(params ...interface{}) string {
	var s []string
	for _, v := range params {
		r := reflect.ValueOf(v)
		if r.Kind() == reflect.Ptr {
			if r.IsNil() {
				continue
			}
			v = r.Elem().Interface()
		}
		s = append(s, fmt.Sprintf("%v", v))
	}
	sort.Strings(s)
	return strings.Join(s, " ")
}

func (d *Client) call(response interface{}, command string, params ...interface{}) error {
	if len(params) == 0 {
		result, err := d.callNoDecode(command)
		if err != nil {
			return err
		}
		return decode(result, response)
	}
	if len(params) == 1 {
		result, err := d.callNoDecode(command, params[0])
		if err != nil {
			return err
		}
		return decode(result, response)
	}
	if len(params) == 2 {
		result, err := d.callNoDecode(command, params[0], params[1])
		if err != nil {
			return err
		}
		return decode(result, response)
	}
	//TODO: It should be possible to have n arguments but it would not work with nested variadic arguments
	return errors.Base("parameter size is greater than 2 which is not supported currently.")
}
