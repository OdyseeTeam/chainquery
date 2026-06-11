package lbrycrd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/lbryio/lbry.go/v2/extras/errors"
)

func GetNewAddress(account string) (btcutil.Address, error) {
	params, err := rawParams(account)
	if err != nil {
		return nil, errors.Err(err)
	}
	result, err := rawRequest("getnewaddress", params)
	if err != nil {
		return nil, errors.Err(err)
	}
	var address string
	err = json.Unmarshal(result, &address)
	if err != nil {
		return nil, errors.Err(err)
	}
	chainParams, err := GetChainParams()
	if err != nil {
		return nil, errors.Err(err)
	}
	return btcutil.DecodeAddress(address, chainParams)
}

func ListUnspentMin(minConf int) ([]btcjson.ListUnspentResult, error) {
	params, err := rawParams(minConf)
	if err != nil {
		return nil, errors.Err(err)
	}
	result, err := rawRequest("listunspent", params)
	if err != nil {
		return nil, errors.Err(err)
	}
	var unspent []btcjson.ListUnspentResult
	err = json.Unmarshal(result, &unspent)
	if err != nil {
		return nil, errors.Err(err)
	}
	return unspent, nil
}

func CreateRawTransaction(inputs []btcjson.TransactionInput, amounts map[btcutil.Address]btcutil.Amount, lockTime *int64) (*wire.MsgTx, error) {
	convertedAmounts := make(map[string]float64, len(amounts))
	for address, amount := range amounts {
		convertedAmounts[address.String()] = amount.ToBTC()
	}
	params, err := rawParams(inputs, convertedAmounts)
	if err != nil {
		return nil, errors.Err(err)
	}
	if lockTime != nil {
		params, err = rawParams(inputs, convertedAmounts, *lockTime)
		if err != nil {
			return nil, errors.Err(err)
		}
	}
	result, err := rawRequest("createrawtransaction", params)
	if err != nil {
		return nil, errors.Err(err)
	}
	return decodeTransactionResult(result)
}

func SignRawTransactionWithWallet(tx *wire.MsgTx) (*wire.MsgTx, bool, error) {
	txHex, err := serializeTransaction(tx)
	if err != nil {
		return nil, false, errors.Err(err)
	}
	params, err := rawParams(txHex)
	if err != nil {
		return nil, false, errors.Err(err)
	}
	result, err := rawRequest("signrawtransactionwithwallet", params)
	if err != nil {
		return nil, false, errors.Err(err)
	}
	var signResult btcjson.SignRawTransactionResult
	err = json.Unmarshal(result, &signResult)
	if err != nil {
		return nil, false, errors.Err(err)
	}
	serializedTx, err := hex.DecodeString(signResult.Hex)
	if err != nil {
		return nil, false, errors.Err(err)
	}
	var msgTx wire.MsgTx
	err = msgTx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return nil, false, errors.Err(err)
	}
	return &msgTx, signResult.Complete, nil
}

func SendRawTransaction(tx *wire.MsgTx, allowHighFees bool) (*chainhash.Hash, error) {
	txHex, err := serializeTransaction(tx)
	if err != nil {
		return nil, errors.Err(err)
	}
	params, err := rawParams(txHex, allowHighFees)
	if err != nil {
		return nil, errors.Err(err)
	}
	result, err := rawRequest("sendrawtransaction", params)
	if err != nil {
		return nil, errors.Err(err)
	}
	var txHash string
	err = json.Unmarshal(result, &txHash)
	if err != nil {
		return nil, errors.Err(err)
	}
	return chainhash.NewHashFromStr(txHash)
}

func rawParams(params ...interface{}) ([]json.RawMessage, error) {
	rawMessages := make([]json.RawMessage, len(params))
	for i, param := range params {
		rawMessage, err := json.Marshal(param)
		if err != nil {
			return nil, errors.Err(err)
		}
		rawMessages[i] = rawMessage
	}
	return rawMessages, nil
}

func decodeTransactionResult(result json.RawMessage) (*wire.MsgTx, error) {
	var txHex string
	err := json.Unmarshal(result, &txHex)
	if err != nil {
		return nil, errors.Err(err)
	}
	serializedTx, err := hex.DecodeString(txHex)
	if err != nil {
		return nil, errors.Err(err)
	}
	var msgTx wire.MsgTx
	err = msgTx.Deserialize(bytes.NewReader(serializedTx))
	if err != nil {
		return nil, errors.Err(err)
	}
	return &msgTx, nil
}

func serializeTransaction(tx *wire.MsgTx) (string, error) {
	if tx == nil {
		return "", nil
	}
	buffer := bytes.NewBuffer(make([]byte, 0, tx.SerializeSize()))
	err := tx.Serialize(buffer)
	if err != nil {
		return "", errors.Err(err)
	}
	return hex.EncodeToString(buffer.Bytes()), nil
}
