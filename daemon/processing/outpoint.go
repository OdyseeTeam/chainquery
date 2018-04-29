package processing

import (
	"encoding/hex"
	"encoding/json"
	"strconv"

	ds "github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	m "github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"

	"github.com/sirupsen/logrus"
)

type vinToProcess struct {
	jsonVin *lbrycrd.Vin
	tx      *m.Transaction
	txDC    *txDebitCredits
}

type voutToProcess struct {
	jsonVout *lbrycrd.Vout
	tx       *m.Transaction
	txDC     *txDebitCredits
}

func initVinWorkers(nrWorkers int, jobs <-chan vinToProcess, results chan<- error) {
	for i := 0; i < nrWorkers; i++ {
		go vinProcessor(jobs, results)
	}
}

func vinProcessor(jobs <-chan vinToProcess, results chan<- error) {
	for job := range jobs {
		results <- processVin(job.jsonVin, job.tx, job.txDC)
	}
}

func initVoutWorkers(nrWorkers int, jobs <-chan voutToProcess, results chan<- error) {
	for i := 0; i < nrWorkers; i++ {
		go voutProcessor(jobs, results)
	}
}

func voutProcessor(jobs <-chan voutToProcess, results chan<- error) {
	for job := range jobs {
		results <- processVout(job.jsonVout, job.tx, job.txDC)
	}
}

func processVin(jsonVin *lbrycrd.Vin, tx *m.Transaction, txDC *txDebitCredits) error {
	vin := &m.Input{}
	foundVin := ds.GetInput(tx.Hash, len(jsonVin.Coinbase) > 0, jsonVin.TxID, uint(jsonVin.Vout))
	if foundVin != nil {
		vin = foundVin
	}
	vin.TransactionID = tx.ID
	vin.TransactionHash = tx.Hash
	vin.Sequence = uint(jsonVin.Sequence)

	if jsonVin.Coinbase != "" { //
		// No Source Output - Generation of Coin
		if err := processCoinBaseVin(jsonVin, vin); err != nil {
			return err
		}
	} else {
		vin.PrevoutHash.String = jsonVin.TxID
		vin.PrevoutHash.Valid = true
		vin.PrevoutN.Uint = uint(jsonVin.Vout)
		vin.PrevoutN.Valid = true
		vin.ScriptSigHex.String = jsonVin.ScriptSig.Hex
		vin.ScriptSigHex.Valid = true
		vin.ScriptSigAsm.String = jsonVin.ScriptSig.Asm
		vin.ScriptSigAsm.Valid = true
		srcOutput := ds.GetOutput(vin.PrevoutHash.String, vin.PrevoutN.Uint)
		if srcOutput == nil {
			id := strconv.Itoa(int(tx.ID))
			sequence := strconv.FormatUint(uint64(vin.Sequence), 10)
			logrus.Error("Tx ", tx.ID, ", Vin ", vin.PrevoutN.Uint, " - ", vin.PrevoutHash.String)
			err := errors.Base("No source output for vin in tx: (" + id + ") - (" + sequence + ")")
			panic(err)
		}
		if srcOutput != nil {

			vin.Value = srcOutput.Value
			var addresses []string
			if srcOutput.AddressList.Valid {
				if err := json.Unmarshal([]byte(srcOutput.AddressList.String), &addresses); err != nil {
					logrus.Error("Error unmarshalling source output address list: ", err)
				}
			}
			var address *m.Address
			if len(addresses) > 0 {
				address = ds.GetAddress(addresses[0])
			} else if srcOutput.Type.String == lbrycrd.NonStandard {

				jsonAddress, err := getAddressFromNonStandardVout(srcOutput.ScriptPubKeyHex.String)
				if err != nil {
					return err
				}
				address = ds.GetAddress(jsonAddress)
				if address == nil {
					logrus.Error("No addresses for vout address list! ", srcOutput.ID, " -> ", srcOutput.AddressList.String)
					panic(nil)
				}

			}
			if address != nil {
				txDC.subtract(address.Address, srcOutput.Value.Float64)
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
			} else {
				logrus.Error("No Address created for Vin: ", vin.ID, " of tx ", tx.ID, " vout: ", srcOutput.ID, " Address: ", addresses[0])
				panic(nil)
			}
			// Update the srcOutput spent if successful
			srcOutput.IsSpent = true
			srcOutput.SpentByInputID.Uint64 = vin.ID
			err := ds.PutOutput(srcOutput)
			if err != nil {
				return err
			}

			//Make sure there is a transaction address

			if ds.GetTxAddress(tx.ID, vin.InputAddressID.Uint64) == nil {
				return errors.Base("Missing txAddress for Tx" + strconv.Itoa(int(tx.ID)) + "- Addr:" + strconv.Itoa(int(vin.InputAddressID.Uint64)))
			}
		}
	}
	return nil
}

func processCoinBaseVin(jsonVin *lbrycrd.Vin, vin *m.Input) error {
	vin.IsCoinbase = true
	vin.Coinbase.String = jsonVin.Coinbase
	vin.Coinbase.Valid = true
	err := ds.PutInput(vin)
	if err != nil {
		return err
	}
	return nil
}

func processVout(jsonVout *lbrycrd.Vout, tx *m.Transaction, txDC *txDebitCredits) error {
	vout := &m.Output{}
	foundVout := ds.GetOutput(tx.Hash, uint(jsonVout.N))
	if foundVout != nil {
		vout = foundVout
	}

	vout.TransactionID = tx.ID
	vout.TransactionHash = tx.Hash
	vout.Vout = uint(jsonVout.N)
	vout.Value.Float64 = jsonVout.Value
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
	} else if vout.Type.String == lbrycrd.NonStandard {
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
	} else {
		//All addresses for transaction are created and inserted into the DB ahead of time
		logrus.Error("No address in db for \"", jsonAddresses[0], "\" txId: ", tx.ID)
		panic(nil)
	}

	// Save output
	err = ds.PutOutput(vout)
	if err != nil {
		return err
	}

	//Make sure there is a transaction address
	if ds.GetTxAddress(tx.ID, address.ID) == nil {
		return errors.Base("Missing txAddress for Tx:" + strconv.Itoa(int(tx.ID)) + "- Addr:" + strconv.Itoa(int(address.ID)))
	}

	// Process script for potential claims
	err = processScript(*vout, *tx)
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

func processScript(vout m.Output, tx m.Transaction) error {
	scriptBytes, err := hex.DecodeString(vout.ScriptPubKeyHex.String)
	if err != nil {
		return err
	}
	isNonStandard := vout.Type.String == lbrycrd.NonStandard
	isClaimScript := lbrycrd.IsClaimScript(scriptBytes)
	if isNonStandard && isClaimScript {
		_, err = processAsClaim(scriptBytes, vout, tx)
		if err != nil {
			return err
		}
	} else if isNonStandard {
		logrus.Error("Non standard script and not a valid claim!")
	}

	return nil
}
