package lbrycrd

import (
	"time"

	"github.com/lbryio/chainquery/util"
)

//GetBlock performs a jsonrpc that returns the structured data as a GetBlockResponse.
//If LBRYcrd contains this block it will be returned.
func GetBlock(blockHash string) (*GetBlockResponse, error) {
	defer util.TimeTrack(time.Now(), "getblock", "lbrycrdprofile")
	response := new(GetBlockResponse)

	return response, call(&response, "getblock", blockHash)
}

//GetBlockHash performs a jsonrpc that returns the hash of the block as a string.
func GetBlockHash(i uint64) (*string, error) {
	defer util.TimeTrack(time.Now(), "getblockhash", "lbrycrdprofile")
	rawresponse, err := callNoDecode("getblockhash", i)
	if err != nil {
		return nil, err
	}
	value := rawresponse.(string)

	return &value, nil
}

// GetBlockCount returns the highest block LBRYcrd is aware of.
func GetBlockCount() (*uint64, error) {
	defer util.TimeTrack(time.Now(), "getblockcount", "lbrycrdprofile")
	rawresponse, err := callNoDecode("getblockcount")
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

// GetRawTransactionResponse returns the raw transactions structured data. This will not always work. LBRYcrd must have
//-txindex turned on otherwise only transactions in the memory pool can be returned.
func GetRawTransactionResponse(hash string) (*TxRawResult, error) {
	defer util.TimeTrack(time.Now(), "getrawtransaction", "lbrycrdprofile")
	response := new(TxRawResult)

	return response, call(&response, "getrawtransaction", hash, 1)
}

// GetBalance returns the balance of a wallet address.
func GetBalance() (*float64, error) {
	defer util.TimeTrack(time.Now(), "getbalance", "lbrycrdprofile")
	rawresponse, err := callNoDecode("getbalance")
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

// GetClaimsInTrie gets all the claims current active in the claim trie
func GetClaimsInTrie() ([]ClaimNameResult, error) {
	defer util.TimeTrack(time.Now(), "getclaimsintrie", "lbrycrdprofile")
	response := new([]ClaimNameResult)

	return *response, call(&response, "getclaimsintrie")
}

// GetClaimsForName gets all the claims for a name in the claimtrie.
func GetClaimsForName(name string) (ClaimsForNameResult, error) {
	response := new(ClaimsForNameResult)

	return *response, call(&response, "getclaimsforname", name)
}

// RawMempoolVerboseResponse models the object of mempool results
type RawMempoolVerboseResponse map[string]GetRawMempoolVerboseResult

// GetRawMempool gets all the transactions in the mempool
func GetRawMempool() (RawMempoolVerboseResponse, error) {
	defer util.TimeTrack(time.Now(), "getrawmempool", "lbrycrdprofile")
	response := new(RawMempoolVerboseResponse)

	return *response, call(&response, "getrawmempool", true)
}
