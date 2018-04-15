package lbrycrd

import (
	"github.com/lbryio/chainquery/util"
	"time"
)

//GetBlock performs a jsonrpc that returns the structured data as a GetBlockResponse.
//If LBRYcrd contains this block it will be returned.
func (c *Client) GetBlock(blockHash string) (*GetBlockResponse, error) {
	defer util.TimeTrack(time.Now(), "getblock", "lbrycrdprofile")
	response := new(GetBlockResponse)

	return response, c.call(&response, "getblock", blockHash)
}

//GetBlockHash performs a jsonrpc that returns the hash of the block as a string.
func (c *Client) GetBlockHash(i uint64) (*string, error) {
	defer util.TimeTrack(time.Now(), "getblockhash", "lbrycrdprofile")
	rawresponse, err := c.callNoDecode("getblockhash", i)
	if err != nil {
		return nil, err
	}
	value := rawresponse.(string)

	return &value, nil
}

//Returns the highest block LBRYcrd is aware of.
func (c *Client) GetBlockCount() (*uint64, error) {
	defer util.TimeTrack(time.Now(), "getblockcount", "lbrycrdprofile")
	rawresponse, err := c.callNoDecode("getblockcount")
	if err != nil {
		return nil, err
	}
	value, err := decodeNumber(rawresponse)
	if err != nil {
		return nil, err
	}
	intValue := uint64(value.IntPart())

	return &intValue, nil

}

//Returns the raw transactions structured data. This will not always work. LBRYcrd must have
//-txindex turned on otherwise only transactions in the memory pool can be returned.
func (c *Client) GetRawTransactionResponse(hash string) (*TxRawResult, error) {
	defer util.TimeTrack(time.Now(), "getrawtransaction", "lbrycrdprofile")
	response := new(TxRawResult)

	return response, c.call(&response, "getrawtransaction", hash, 1)
}

//Returns the balance of a wallet address.
func (c *Client) GetBalance(s string) (*float64, error) {
	defer util.TimeTrack(time.Now(), "getbalance", "lbrycrdprofile")
	rawresponse, err := c.callNoDecode("getblance")
	if err != nil {
		return nil, err
	}
	value, err := decodeNumber(rawresponse)
	if err != nil {
		return nil, err
	}
	floatValue, _ := value.Float64()

	return &floatValue, nil
}

//Gets all the claims current active in the claim trie
func (c *Client) GetClaimsInTrie() ([]ClaimNameResult, error) {
	defer util.TimeTrack(time.Now(), "getclaimsintrie", "lbrycrdprofile")
	response := new([]ClaimNameResult)

	return *response, c.call(&response, "getclaimsintrie")
}

func (c *Client) GetClaimsForName(name string) (ClaimsForNameResult, error) {
	response := new(ClaimsForNameResult)

	return *response, c.call(&response, "getclaimsforname", name)
}
