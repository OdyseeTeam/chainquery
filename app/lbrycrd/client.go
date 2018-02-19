package lbrycrd

import (
	"net/url"

	"fmt"
	//"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/lbryio/errors.go"
	"github.com/mitchellh/mapstructure"
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

// jsonRequest holds information about a json request that is used to properly
// detect, interpret, and deliver a reply to it.
type jsonRequest struct {
	id             uint64
	method         string
	cmd            interface{}
	marshalledJSON []byte
	responseChan   chan *response
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
	println("debug number params", len(params))
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
	println("number of params1", len(params))
	var p []interface{}
	if len(params) != 0 {
		p = params
	}
	println("number of params2", len(p))
	result, err := d.callNoDecode(command, p)
	if err != nil {
		return err
	}
	return decode(result, response)
}

func (c *Client) GetBlock(blockHash string) (*GetBlockResponse, error) {
	response := new(GetBlockResponse)
	//hash, _ := chainhash.NewHashFromStr(blockHash)
	return response, c.call(response, "getblock", "MarkieMark")
}
func (c *Client) GetBlockHash(i int64) (string, error) {
	//return "b6d31e06e38debfcb906ff262fd786c3a8dbd5cdc81469cf7a7d89ebe9257b5a", nil
	var response string = ""
	return response, c.call(response, "getblockhash", i)
}
func (client *Client) Shutdown() {
	client.Shutdown()
}
func (c *Client) GetBlockCount() (int64, error) {
	var response int64 = 0
	return response, c.call(response, "getblockcount")
}
func (client *Client) GetRawTransactionResponse(hash string) (*TxRawResult, error) {
	return nil, nil
}
