package processing

import (
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/extras/errors"

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
			foundAddress, _ := model.Addresses(qm.Where(model.AddressColumns.Address+"=?", address)).OneG()
			if foundAddress != nil {
				addressIDMap[address] = foundAddress.ID
				if foundAddress.FirstSeen.Valid && foundAddress.FirstSeen.Time.Unix() == 0 {
					foundAddress.FirstSeen.SetValid(time.Unix(int64(blockSeconds), 0))
					err := datastore.PutAddress(foundAddress)
					if err != nil {
						return nil, errors.Err(err)
					}
				}
				err := createTxAddressIfNotExist(tx.ID, foundAddress.ID)
				if err != nil {
					return nil, err
				}
			} else {
				newAddress := model.Address{}
				newAddress.Address = address
				newAddress.FirstSeen.SetValid(time.Unix(int64(blockSeconds), 0))
				err := datastore.PutAddress(&newAddress)
				if err != nil {
					return nil, err
				}
				addressIDMap[address] = newAddress.ID
				err = createTxAddressIfNotExist(tx.ID, newAddress.ID)
				if err != nil {
					return nil, err
				}
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
			return nil, errors.Base("Missing source output for " + input.TxID + "-" + strconv.Itoa(int(input.Vout)))
		}
		var addresses []string
		if !srcOutput.AddressList.Valid {
			jsonAddress, err := getAddressFromNonStandardVout(srcOutput.ScriptPubKeyHex.String)
			if err != nil {
				return nil, errors.Prefix("AddressParseError: ", err)
			}
			addresses = append(addresses, jsonAddress)
		} else {
			err := json.Unmarshal([]byte(srcOutput.AddressList.String), &addresses)
			if err != nil {
				logrus.Panic(errors.Prefix("Could not parse AddressList from source output", err))
			}
			if len(addresses) == 0 {
				return nil, errors.Err("No addresses were found for inputs of %s", tx.Hash)
			}
		}
		for _, address := range addresses {
			addr := datastore.GetAddress(address)
			addressIDMap[address] = addr.ID
			err := createTxAddressIfNotExist(tx.ID, addr.ID)
			if err != nil {
				return nil, err
			}

		}
	}
	return addressIDMap, nil

}

func createTxAddressIfNotExist(txID uint64, addressID uint64) error {
	if datastore.GetTxAddress(txID, addressID) == nil {
		txAddress := model.TransactionAddress{}
		txAddress.TransactionID = txID
		txAddress.AddressID = addressID
		if err := datastore.PutTxAddress(&txAddress); err != nil {
			return errors.Err(err) //Should never happen.
		}
	}
	return nil
}
