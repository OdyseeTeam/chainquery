package processing

import (
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	ds "github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	m "github.com/lbryio/chainquery/model"
	"github.com/lbryio/errors.go"

	"github.com/sirupsen/logrus"
)

func ProcessVin(jsonVin *lbrycrd.Vin, txId *uint64, txHash string, txDC txDebitCredits) error {
	vin := &m.Input{}
	foundVin := ds.GetInput(txHash, len(jsonVin.Coinbase) > 0, jsonVin.Txid, uint(jsonVin.Vout))
	if foundVin != nil {
		vin = foundVin
	}
	vin.TransactionID = *txId
	vin.TransactionHash = txHash
	vin.Sequence = uint(jsonVin.Sequence)

	if jsonVin.Coinbase != "" { //
		// No Source Output - Generation of Coin
		err := processCoinBaseVin(jsonVin, vin)
		return err
	} else {
		vin.PrevoutHash.String = jsonVin.Txid
		vin.PrevoutHash.Valid = true
		vin.PrevoutN.Uint = uint(jsonVin.Vout)
		vin.PrevoutN.Valid = true
		vin.ScriptSigHex.String = jsonVin.ScriptSig.Hex
		vin.ScriptSigHex.Valid = true
		vin.ScriptSigAsm.String = jsonVin.ScriptSig.Asm
		vin.ScriptSigAsm.Valid = true
		src_output := ds.GetOutput(vin.PrevoutHash.String, vin.PrevoutN.Uint)
		if src_output == nil {
			id := strconv.Itoa(int(*txId))
			sequence := strconv.FormatUint(uint64(vin.Sequence), 10)
			logrus.Error("Tx ", *txId, ", Vin ", vin.PrevoutN.Uint, " - ", vin.PrevoutHash.String)
			err := errors.Base("No source output for vin in tx: (" + id + ") - (" + sequence + ")")
			panic(err)
		}
		if src_output != nil {
			vin.Value = src_output.Value
			var addresses []string
			json.Unmarshal([]byte(src_output.AddressList.String), &addresses)
			var address *m.Address
			if len(addresses) > 0 {
				address = ds.GetAddress(addresses[0])
			} else if src_output.Type.String == lbrycrd.NON_STANDARD {
				jsonAddress, err := getAddressFromNonStandardVout(src_output.ScriptPubKeyHex.String)
				if err != nil {
					return err
				}
				address = ds.GetAddress(jsonAddress)
				if address == nil {
					logrus.Error("No addresses for vout address list! ", src_output.ID, " -> ", src_output.AddressList.String)
					panic(nil)
				}
			}
			if address != nil {
				value, _ := strconv.ParseFloat(src_output.Value.String, 64)
				txDC.subtract(address.Address, value)
				vin.InputAddressID.Uint64 = address.ID
				vin.InputAddressID.Valid = true
				// Store input - Needed to store input address below
				err := ds.PutInput(vin)
				if err != nil {
					return err
				}
				err = vin.SetInputAddressG(false, address)
				if err != nil {
					logrus.Error("Failure inserting InputAddress: Vin ", vin.ID, "Address(", address.ID, ") ", address.Address)
					panic(err)
				}
				err = vin.SetAddressesG(false, address)
				if err != nil {
					logrus.Error("Failure adding addresses: Vin ", vin.ID, ", Tx ", *txId, ", Vout ", src_output.ID, ", Address(", address.ID, ") ", address.Address)
					panic(err)
				}
			} else {
				logrus.Error("No Address created for Vin: ", vin.ID, " of tx ", *txId, " vout: ", src_output.ID, " Address: ", addresses[0])
				panic(nil)
			}

			// Update the src_output spent if successful
			src_output.IsSpent = true
			src_output.SpentByInputID.Uint64 = vin.ID
			err := ds.PutOutput(src_output)
			if err != nil {
				return err
			}

			//Make sure there is a transaction address
			txAddress := createTransactionAddress(*txId, vin.InputAddressID.Uint64)

			err = ds.PutTxAddress(&txAddress)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func processCoinBaseVin(jsonVin *lbrycrd.Vin, vin *m.Input) error {
	//log.Debug("Coinbase transaction")//
	vin.IsCoinbase = true
	vin.Coinbase.String = jsonVin.Coinbase
	vin.Coinbase.Valid = true
	err := ds.PutInput(vin)
	if err != nil {
		return err
	}
	return nil
}

func ProcessVout(jsonVout *lbrycrd.Vout, txId *uint64, txHash string, txDC txDebitCredits) error {
	vout := &m.Output{}
	foundVout := ds.GetOutput(txHash, uint(jsonVout.N))
	if foundVout != nil {
		vout = foundVout
	}

	vout.TransactionID = *txId
	vout.TransactionHash = txHash
	vout.Vout = uint(jsonVout.N)
	value := strconv.FormatFloat(jsonVout.Value, 'f', -1, 64)
	vout.Value.String = value
	vout.Value.Valid = true
	vout.ScriptPubKeyAsm.String = jsonVout.ScriptPubKey.Asm
	vout.ScriptPubKeyAsm.Valid = true
	vout.ScriptPubKeyHex.String = jsonVout.ScriptPubKey.Hex
	vout.ScriptPubKeyHex.Valid = true
	vout.Type.String = jsonVout.ScriptPubKey.Type
	vout.Type.Valid = true
	jsonAddresses, err := json.Marshal(jsonVout.ScriptPubKey.Addresses)
	var address *m.Address
	if len(jsonVout.ScriptPubKey.Addresses) > 0 {
		address = ds.GetAddress(jsonVout.ScriptPubKey.Addresses[0])
		vout.AddressList.String = string(jsonAddresses)
		vout.AddressList.Valid = true
	} else if vout.Type.String == lbrycrd.NON_STANDARD {
		jsonAddress, err := getAddressFromNonStandardVout(vout.ScriptPubKeyHex.String)
		if err != nil {
			return err
		}
		address = ds.GetAddress(jsonAddress)
	}
	if err != nil {
		logrus.Error("Could not marshall address list of Vout")
		err = nil //reset error/
	} else if address != nil {
		txDC.add(address.Address, jsonVout.Value)
		vout.SetAddressesG(false, address)
	} else {
		//All addresses for transaction are created and inserted into the DB ahead of time
		logrus.Error("No address in db for \"", jsonAddresses[0], "\" txId: ", *txId)
		panic(nil)
	}

	// Save output
	err = ds.PutOutput(vout)
	if err != nil {
		return err
	}
	//Make sure there is a transaction address
	txAddress := createTransactionAddress(*txId, address.ID)
	err = ds.PutTxAddress(&txAddress)
	if err != nil {
		return err
	}

	// Process script for potential claims
	err = processScript(*vout)
	if err != nil {
		return err
	}

	return nil
}

func getAddressFromNonStandardVout(hexString string) (address string, err error) {
	scriptBytes, err := hex.DecodeString(hexString)
	if err != nil {
		return "", err
	}
	pksBytes, err := lbrycrd.GetPubKeyScriptFromClaimPKS(scriptBytes)
	if err != nil {
		return "", err
	}
	address = lbrycrd.GetAddressFromPublicKeyScript(pksBytes)
	return address, nil
}

func createTransactionAddress(txID uint64, addressID uint64) m.TransactionAddress {

	txAddress := m.TransactionAddress{}
	txAddress.TransactionID = txID
	txAddress.AddressID = addressID
	txAddress.DebitAmount = "0.0"
	txAddress.CreditAmount = "0.0"
	txAddress.LatestTransactionTime = time.Now()

	return txAddress
}

func processScript(vout m.Output) error {
	/*scriptBytes, err := hex.DecodeString(vout.ScriptPubKeyHex.String)
	if err != nil {
		return err
	}
	isNonStandard := vout.Type.String == lbrycrd.NON_STANDARD
	if isNonStandard {
		err = processAsClaim(scriptBytes, vout)
		if err != nil {
			return err
		}
	}
	*/
	return nil
}
