package processing

import (
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

func createUpdateVoutAddresses(tx *model.Transaction, outputs *[]lbrycrd.Vout, blockSeconds uint64) (map[string]uint64, error) {
	addressIDMap := make(map[string]uint64)
	for _, output := range *outputs {
		if len(output.ScriptPubKey.Addresses) == 0 {
			if output.ScriptPubKey.Type == lbrycrd.NonStandard {
				scriptBytes, err := hex.DecodeString(output.ScriptPubKey.Hex)
				if err != nil {
					return nil, err
				}
				if lbrycrd.IsClaimScript(scriptBytes) {
					pksBytes, err := lbrycrd.GetPubKeyScriptFromClaimPKS(scriptBytes)
					if err != nil {
						return nil, err
					}
					address := lbrycrd.GetAddressFromPublicKeyScript(pksBytes)
					addrSet := append(output.ScriptPubKey.Addresses, address)
					output.ScriptPubKey.Addresses = addrSet
				}
			}
		}
		for _, address := range output.ScriptPubKey.Addresses {
			foundAddress, _ := model.AddressesG(qm.Where(model.AddressColumns.Address+"=?", address)).One()
			if foundAddress != nil {
				addressIDMap[address] = foundAddress.ID
				createTxAddressIfNotExist(tx.ID, foundAddress.ID)
			} else {
				newAddress := model.Address{}
				newAddress.Address = address
				newAddress.FirstSeen.Time = time.Unix(int64(blockSeconds), 0)
				newAddress.FirstSeen.Valid = true
				err := datastore.PutAddress(&newAddress)
				if err != nil {
					return nil, err
				}
				addressIDMap[address] = newAddress.ID
				createTxAddressIfNotExist(tx.ID, newAddress.ID)
			}
		}
	}

	return addressIDMap, nil

}

func createUpdateVinAddresses(tx *model.Transaction, inputs *[]lbrycrd.Vin, blockSeconds uint64) (map[string]uint64, error) {
	addressIDMap := make(map[string]uint64)
	for _, input := range *inputs {
		srcOutput := datastore.GetOutput(input.TxID, uint(input.Vout))
		if srcOutput == nil {
			if input.Coinbase != "" {
				continue //No addresses for coinbase inputs.
			}
			logrus.Warning("Missing source output for " + input.TxID + "-" + strconv.Itoa(int(input.Vout)) + ": attempting to fix...")
			//Attempt to fix automatically
			err := fixMissingSourceOutput(input.TxID)
			if err != nil {
				return nil, errors.Prefix("could not fix missing source output for "+input.TxID+"-"+strconv.Itoa(int(input.Vout))+" due to: ", err)
			}
			srcOutput = datastore.GetOutput(input.TxID, uint(input.Vout))
			if srcOutput != nil {
				return nil, errors.Base("Missing source output for " + input.TxID + "-" + strconv.Itoa(int(input.Vout)))
			}
		}
		var addresses []string
		if !srcOutput.AddressList.Valid {
			jsonAddress, err := getAddressFromNonStandardVout(srcOutput.ScriptPubKeyHex.String)
			if err != nil {
				logrus.Error("AddressParseError: ", err)
			}
			addresses = append(addresses, jsonAddress)
		} else {
			err := json.Unmarshal([]byte(srcOutput.AddressList.String), &addresses)
			if err != nil {
				panic(errors.Prefix("Could not parse AddressList from source output", err))
			}
		}
		for _, address := range addresses {
			addr := datastore.GetAddress(address)
			addressIDMap[address] = addr.ID
			createTxAddressIfNotExist(tx.ID, addr.ID)
		}
	}
	return addressIDMap, nil

}

func createTxAddressIfNotExist(txID uint64, addressID uint64) {
	if datastore.GetTxAddress(txID, addressID) == nil {
		txAddress := model.TransactionAddress{}
		txAddress.TransactionID = txID
		txAddress.AddressID = addressID
		txAddress.LatestTransactionTime = time.Now()
		if err := datastore.PutTxAddress(&txAddress); err != nil {
			panic(err) //Should never happen.
		}
	}
}
