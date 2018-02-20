package lbrycrd

import (
	"net/url"

	"fmt"
	"github.com/lbryio/errors.go"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/ybbus/jsonrpc"
	"reflect"
	"sort"
	"strings"
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

// New initializes a new Client
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

func (c *Client) GetBlockHeight() int64 {
	/*count, err := c.Client.GetBlockCount()
	if err != nil {
		return -1
	}*/
	return -1
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
	logrus.Error("parameter size is greater than 1")
	return nil
}

func (c *Client) GetBlock(blockHash string) (*GetBlockResponse, error) {
	response := new(GetBlockResponse)
	return response, c.call(&response, "getblock", blockHash)
}
func (c *Client) GetBlockHash(i uint64) (*string, error) {
	rawresponse, err := c.callNoDecode("getblockhash", i)
	if err != nil {
		return nil, err
	}
	value := rawresponse.(string)
	logrus.Debug("GetBlockHashResponse ", value)
	return &value, nil
}
func (client *Client) Shutdown() {
	client.Shutdown()
}

func (c *Client) GetBlockCount() (*uint64, error) {
	rawresponse, err := c.callNoDecode("getblockcount")
	if err != nil {
		return nil, err
	}
	value, err := decodeNumber(rawresponse)
	if err != nil {
		return nil, err
	}
	intValue := uint64(value.IntPart())
	logrus.Debug("GetBlockCountResult ", intValue)
	return &intValue, nil

}
func (c *Client) GetRawTransactionResponse(hash string) (*TxRawResult, error) {
	response := new(TxRawResult)
	return response, c.call(&response, "getrawtransaction", hash, 1)
}
func (c *Client) GetBalance(s string) (*float64, error) {
	rawresponse, err := c.callNoDecode("getblance")
	if err != nil {
		return nil, err
	}
	value, err := decodeNumber(rawresponse)
	if err != nil {
		return nil, err
	}
	floatValue, _ := value.Float64()
	logrus.Debug("getbalance ", floatValue)
	return &floatValue, nil
}
