package lbrycrd

import (
	"time"

	"github.com/lbryio/chainquery/util"
	"github.com/lbryio/lbry.go/extras/errors"
)

//GetGenesisBlock performs a jsonrpc that returns the structured data as a GetBlockResponse.
//If LBRYcrd contains this block it will be returned.
func GetGenesisBlock() (*GetBlockVerboseResponse, *GetBlockResponse, error) {
	genesisHash, err := GetBlockHash(0)
	if err != nil {
		return nil, nil, errors.Err(err)
	}
	defer util.TimeTrack(time.Now(), "getblock", "lbrycrdprofile")
	verboseResponse := new(GetBlockVerboseResponse)
	response := new(GetBlockResponse)

	err = call(&verboseResponse, "getblock", genesisHash, 2)
	if err != nil {
		return nil, nil, errors.Err(err)
	}

	return verboseResponse, response, call(&response, "getblock", genesisHash)
}

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

// ClaimName creates a claim transaction for lbrycrd.
func ClaimName(name string, hexValue string, amount float64) (string, error) {
	defer util.TimeTrack(time.Now(), "claimname", "lbrycrdprofile")

	rawresponse, err := callNoDecode("claimname", name, hexValue, amount)
	if err != nil {
		return "", err
	}

	value, ok := rawresponse.(string)
	if !ok {
		return "", errors.Err("response is not a string")
	}

	return value, nil
}

//GenerateBlocks generates n blocks in regtest. Will error in mainnet or testnet.
func GenerateBlocks(count int64) ([]string, error) {
	defer util.TimeTrack(time.Now(), "generate", "lbrycrdprofile")
	var response []string
	return response, call(&response, "generate", count)
}
