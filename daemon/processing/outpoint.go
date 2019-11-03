package processing

import (
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"

	ds "github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	m "github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/extras/stop"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
)

type vinToProcess struct {
	jsonVin *lbrycrd.Vin
	tx      *m.Transaction
	txDC    *txDebitCredits
	vin     uint64
}

type voutToProcess struct {
	jsonVout    *lbrycrd.Vout
	tx          *m.Transaction
	txDC        *txDebitCredits
	blockHeight uint64
}

func initVinWorkers(s *stop.Group, nrWorkers int, jobs <-chan vinToProcess, results chan<- error) {
	for i := 0; i < nrWorkers; i++ {
		s.Add(1)
		go func(worker int) {
			defer s.Done()
			vinProcessor(worker, jobs, results)
		}(i)
	}
}

func vinProcessor(worker int, jobs <-chan vinToProcess, results chan<- error) {
	for job := range jobs {
		q(strconv.Itoa(worker) + " - WORKER VIN start new job " + strconv.Itoa(int(job.jsonVin.Sequence)))
		result := processVin(job.jsonVin, job.tx, job.txDC, job.vin)
		q(strconv.Itoa(worker) + " - WORKER VIN passing result " + strconv.Itoa(int(job.jsonVin.Sequence)))
		results <- result
		q(strconv.Itoa(worker) + " - WORKER VIN passed result " + strconv.Itoa(int(job.jsonVin.Sequence)))
	}
	q(strconv.Itoa(worker) + " - WORKER VIN finished all jobs")
}

func initVoutWorkers(s *stop.Group, nrWorkers int, jobs <-chan voutToProcess, results chan<- error) {
	for i := 0; i < nrWorkers; i++ {
		s.Add(1)
		go func(worker int) {
			defer s.Done()
			voutProcessor(worker, jobs, results)
		}(i)
	}
}

func voutProcessor(worker int, jobs <-chan voutToProcess, results chan<- error) {
	for job := range jobs {
		results <- processVout(job.jsonVout, job.tx, job.txDC, job.blockHeight)
	}
	q(strconv.Itoa(worker) + " - WORKER VOUT finished all jobs")
}

func processVin(jsonVin *lbrycrd.Vin, tx *m.Transaction, txDC *txDebitCredits, n uint64) error {
	vin := &m.Input{}
	foundVin := ds.GetInput(tx.Hash, len(jsonVin.Coinbase) > 0, jsonVin.TxID, uint(jsonVin.Vout))
	if foundVin != nil {
		vin = foundVin
	}
	vin.Vin.SetValid(uint(n))
	vin.TransactionID = tx.ID
	vin.TransactionHash = tx.Hash
	vin.Sequence = uint(jsonVin.Sequence)
	vin.Witness.String = strings.Join(jsonVin.Witness, ",")

	if jsonVin.Coinbase != "" { //
		// No Source Output - Generation of Coin
		if err := processCoinBaseVin(jsonVin, vin); err != nil {
			return err
		}
	} else {
		vin.PrevoutHash.SetValid(jsonVin.TxID)
		vin.PrevoutN.SetValid(uint(jsonVin.Vout))
		vin.ScriptSigHex.SetValid(jsonVin.ScriptSig.Hex)
		vin.ScriptSigAsm.SetValid(jsonVin.ScriptSig.Asm)
		srcOutput := ds.GetOutput(vin.PrevoutHash.String, vin.PrevoutN.Uint)
		if srcOutput == nil {
			id := strconv.Itoa(int(tx.ID))
			sequence := strconv.FormatUint(uint64(vin.Sequence), 10)
			logrus.Error("Tx ", tx.ID, ", Vin ", vin.PrevoutN.Uint, " - ", vin.PrevoutHash.String)
			err := errors.Base("No source output for vin in tx: (" + id + ") - (" + sequence + ")")
			return err
		}
		if srcOutput != nil {
			vin.Value = srcOutput.Value
			var addresses []string
			if srcOutput.AddressList.Valid {
				if err := json.Unmarshal([]byte(srcOutput.AddressList.String), &addresses); err != nil {
					return errors.Err("Error unmarshalling source output address list: ", err)
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
					return errors.Err("No addresses for vout address list! %d -> %s ", srcOutput.ID, srcOutput.AddressList.String)
				}

			}
			if address != nil {
				txDC.subtract(address.Address, srcOutput.Value.Float64)
				vin.InputAddressID.SetValid(address.ID)
				// Store input - Needed to store input address below
				err := ds.PutInput(vin)
				if err != nil {
					return err
				}
			} else {
				return errors.Err("No Address created for Vin: %d of tx %d vout: %d Address: %s", vin.ID, tx.ID, srcOutput.ID, addresses[0])
			}
			// Update the srcOutput spent if successful
			srcOutput.IsSpent = true
			srcOutput.SpentByInputID.SetValid(vin.ID)
			c := m.OutputColumns
			err := ds.PutOutput(srcOutput, boil.Whitelist(c.IsSpent, c.SpentByInputID))
			if err != nil {
				return err
			}

			//Make sure there is a transaction address

			if ds.GetTxAddress(tx.ID, vin.InputAddressID.Uint64) == nil {
				return errors.Err("Missing txAddress for Tx: " + strconv.Itoa(int(tx.ID)) + " - Addr: " + strconv.Itoa(int(vin.InputAddressID.Uint64)) + "[" + address.Address + "]")
			}
		}
	}
	return nil
}

func processCoinBaseVin(jsonVin *lbrycrd.Vin, vin *m.Input) error {
	vin.IsCoinbase = true
	vin.Coinbase.SetValid(jsonVin.Coinbase)
	err := ds.PutInput(vin)
	if err != nil {
		return err
	}
	return nil
}

func processVout(jsonVout *lbrycrd.Vout, tx *m.Transaction, txDC *txDebitCredits, blockHeight uint64) error {
	vout := &m.Output{}
	foundVout := ds.GetOutput(tx.Hash, uint(jsonVout.N))
	if foundVout != nil {
		vout = foundVout
	}

	vout.TransactionID = tx.ID
	vout.TransactionHash = tx.Hash
	vout.Vout = uint(jsonVout.N)
	vout.Value.SetValid(jsonVout.Value)
	vout.RequiredSignatures.SetValid(uint(jsonVout.ScriptPubKey.ReqSigs))
	vout.ScriptPubKeyAsm.SetValid(jsonVout.ScriptPubKey.Asm)
	vout.ScriptPubKeyHex.SetValid(jsonVout.ScriptPubKey.Hex)
	vout.Type.SetValid(jsonVout.ScriptPubKey.Type)
	var address *m.Address
	jsonAddresses, err := json.Marshal(jsonVout.ScriptPubKey.Addresses)
	if len(jsonVout.ScriptPubKey.Addresses) > 0 {
		address = ds.GetAddress(jsonVout.ScriptPubKey.Addresses[0])
		vout.AddressList.SetValid(string(jsonAddresses))
	} else {
		scriptAddress, err := getFirstAddressFromVout(*jsonVout)
		if err != nil {
			return err
		}
		if len(scriptAddress) == 0 {
			return ds.PutOutput(vout, boil.Infer())
		}
		address = ds.GetAddress(scriptAddress)
		vout.AddressList.SetValid(`["` + scriptAddress + `"]`)
	}
	if err != nil {
		logrus.Error("Could not marshall address list of Vout")
		err = nil //reset error/
	} else if address != nil {
		txDC.add(address.Address, jsonVout.Value)
	} else {
		//All addresses for transaction are created and inserted into the DB ahead of time
		return errors.Err("No address in db for \"", jsonAddresses[0], "\" txId: ", tx.ID)
	}

	// Save output
	err = ds.PutOutput(vout, boil.Infer())
	if err != nil {
		return err
	}

	//Make sure there is a transaction address
	if ds.GetTxAddress(tx.ID, address.ID) == nil {
		return errors.Base("Missing txAddress for Tx:" + strconv.Itoa(int(tx.ID)) + "- Addr:" + strconv.Itoa(int(address.ID)))
	}

	// Process script for potential claims
	claimid, err := processScriptForClaim(*vout, *tx, blockHeight)
	if err != nil {
		return err
	}
	if claimid != nil {
		//Update output to link to the proper claim id
		claim := ds.GetClaim(*claimid)
		if claim != nil {
			vout.ClaimID.SetValid(claim.ClaimID)
		}
		// Save output with claim_id
		err = ds.PutOutput(vout, boil.Infer())
		if err != nil {
			return err
		}
	}

	return nil
}

func getAddressFromNonStandardVout(hexString string) (address string, err error) {
	scriptBytes, err := hex.DecodeString(hexString)
	if err != nil {
		return "", errors.Err(err)
	}
	pksBytes, err := lbrycrd.GetPubKeyScriptFromClaimPKS(scriptBytes)
	if err != nil {
		return "", err
	}
	address = lbrycrd.GetAddressFromPublicKeyScript(pksBytes)
	return address, nil
}

func processScriptForClaim(vout m.Output, tx m.Transaction, blockHeight uint64) (*string, error) {
	var claimid *string
	scriptBytes, err := hex.DecodeString(vout.ScriptPubKeyHex.String)
	if err != nil {
		return nil, err
	}
	isNonStandard := vout.Type.String == lbrycrd.NonStandard
	isClaimScript := lbrycrd.IsClaimScript(scriptBytes)
	if isNonStandard && isClaimScript {
		_, claimid, err = processAsClaim(scriptBytes, vout, tx, blockHeight)
		if err != nil {
			return nil, err
		}
	} else if isNonStandard {
		logrus.Error("Non standard script and not a valid claim!")
	}

	return claimid, nil
}
