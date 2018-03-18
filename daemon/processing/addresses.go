package processing

import (
	"encoding/hex"
	"time"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/volatiletech/sqlboiler/queries/qm"
)

func CreateUpdateAddresses(outputs []lbrycrd.Vout, blockSeconds uint64) (map[string]uint64, error) {
	addressIdMap := make(map[string]uint64)
	for _, output := range outputs {
		if len(output.ScriptPubKey.Addresses) == 0 {
			if output.ScriptPubKey.Type == lbrycrd.NON_STANDARD {
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
				addressIdMap[address] = foundAddress.ID
			} else {
				newAddress := model.Address{}
				newAddress.Address = address
				newAddress.FirstSeen.Time = time.Unix(int64(blockSeconds), 0)
				newAddress.FirstSeen.Valid = true
				newAddress.TotalSent = "0.0"
				newAddress.TotalReceived = "0.0"
				err := datastore.PutAddress(&newAddress)
				if err != nil {
					return nil, err
				}
				addressIdMap[address] = newAddress.ID
			}
		}
	}

	return addressIdMap, nil

}
