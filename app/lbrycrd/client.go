package lbrycrd

import (
	"net/url"

	"github.com/lbryio/errors.go"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	btcrpcclient "github.com/btcsuite/btcd/rpcclient" //
	"github.com/btcsuite/btcutil"
)

// MainNetParams define the lbrycrd network. See https://github.com/lbryio/lbrycrd/blob/master/src/chainparams.cpp
var MainNetParams = chaincfg.Params{
	PubKeyHashAddrID: 0x55,
	ScriptHashAddrID: 0x7a,
	PrivateKeyID:     0x1c,
}

func init() {
	// Register lbrycrd network
	err := chaincfg.Register(&MainNetParams)
	if err != nil {
		panic("failed to register lbrycrd network: " + err.Error())
	}
}

// Client connects to a lbrycrd instance
type Client struct {
	*btcrpcclient.Client
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

	connCfg := &btcrpcclient.ConnConfig{
		Host:         u.Host,
		User:         u.User.Username(),
		Pass:         password,
		HTTPPostMode: true, // Bitcoin core only supports HTTP POST mode
		DisableTLS:   true, // Bitcoin core does not provide TLS by default
	}
	// Notice the notification parameter is nil since notifications are not supported in HTTP POST mode.
	client, err := btcrpcclient.New(connCfg, nil)
	if err != nil {
		return nil, errors.Err(err)
	}

	return &Client{client}, nil
}

var errInsufficientFunds = errors.Base("Our wallet is running low. We've been notified, and we will refill it ASAP. Please try again in a little while, or email us at hello@lbry.io for more info.")

// SimpleSend is a convenience function to send credits to an address (0 min confirmations)
func (c *Client) SimpleSend(toAddress string, amount float64) (*chainhash.Hash, error) {
	decodedAddress, err := btcutil.DecodeAddress(toAddress, &MainNetParams)
	if err != nil {
		return nil, errors.Err(err)
	}

	lbcAmount, err := btcutil.NewAmount(amount)
	if err != nil {
		return nil, errors.Err(err)
	}

	hash, err := c.Client.SendFromMinConf("", decodedAddress, lbcAmount, 0)
	if err != nil && err.Error() == "-6: Insufficient funds" {
		err = errors.Err(errInsufficientFunds)
	}
	return hash, errors.Err(err)
}
