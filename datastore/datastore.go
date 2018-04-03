package datastore

import (
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

//Outputs
func GetOutput(txHash string, vout uint) *model.Output {
	//defer util.TimeTrack(time.Now(), "GetOutput")
	txHashMatch := qm.Where(model.OutputColumns.TransactionHash+"=?", txHash)
	vOutMatch := qm.And(model.OutputColumns.Vout+"=?", vout)

	if model.OutputsG(txHashMatch, vOutMatch).ExistsP() {
		output, err := model.OutputsG(txHashMatch, vOutMatch).One()
		if err != nil {
			logrus.Error("Datastore(GETOUTPUT): ", err)
		}
		return output
	}

	return nil
}

func PutOutput(output *model.Output) error {
	//defer util.TimeTrack(time.Now(), "PutOutput")
	if output != nil {
		txHashMatch := qm.Where(model.OutputColumns.TransactionHash+"=?", output.TransactionHash)
		vOutMatch := qm.And(model.OutputColumns.Vout+"=?", output.Vout)
		var err error
		if model.OutputsG(txHashMatch, vOutMatch).ExistsP() {
			err = output.UpdateG()
		} else {
			err = output.InsertG()

		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTOUTPUT): ", err)
			return err
		}
	}

	return nil
}

//Inputs
func GetInput(txHash string, isCoinBase bool, prevHash string, prevN uint) *model.Input {
	//defer util.TimeTrack(time.Now(), "GetInput")
	//Unique
	txHashMatch := qm.Where(model.InputColumns.TransactionHash+"=?", txHash)
	txCoinBaseMatch := qm.Where(model.InputColumns.IsCoinbase+"=?", isCoinBase)
	prevHashMatch := qm.Where(model.InputColumns.PrevoutHash+"=?", prevHash)
	prevNMatch := qm.And(model.InputColumns.PrevoutN+"=?", prevN)

	if model.InputsG(txHashMatch, txCoinBaseMatch, prevHashMatch, prevNMatch).ExistsP() {
		input, err := model.InputsG(txHashMatch, txCoinBaseMatch, prevHashMatch, prevNMatch).One()
		if err != nil {
			logrus.Error("Datastore(GETINPUT): ", err)
		}
		return input
	}

	return nil
}

func PutInput(input *model.Input) error {
	//defer util.TimeTrack(time.Now(), "PutOutput")
	if input != nil {
		//Unique
		txHashMatch := qm.Where(model.InputColumns.TransactionHash+"=?", input.TransactionHash)
		txCoinBaseMatch := qm.Where(model.InputColumns.IsCoinbase+"=?", input.IsCoinbase)
		prevHashMatch := qm.Where(model.InputColumns.PrevoutHash+"=?", input.PrevoutHash)
		prevNMatch := qm.And(model.InputColumns.PrevoutN+"=?", input.PrevoutN)

		var err error
		if model.InputsG(txHashMatch, txCoinBaseMatch, prevHashMatch, prevNMatch).ExistsP() {
			err = input.UpdateG()
		} else {
			err = input.InsertG()

		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTINPUT): ", err)
			return err
		}
	}

	return nil
}

//Addresses

func GetAddress(addr string) *model.Address {
	//defer util.TimeTrack(time.Now(), "GetAddress")
	addrMatch := qm.Where(model.AddressColumns.Address+"=?", addr)

	if model.AddressesG(addrMatch).ExistsP() {

		address, err := model.AddressesG(addrMatch).One()
		if err != nil {
			logrus.Error("Datastore(GETADDRESS): ", err)
		}
		return address
	}

	return nil
}

func PutAddress(address *model.Address) error {
	//defer util.TimeTrack(time.Now(), "PutAddress")
	if address != nil {

		var err error
		if model.AddressExistsGP(address.ID) {
			err = address.UpdateG()
		} else {
			err = address.InsertG()

		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTADDRESS): ", err)
			return err
		}
	}

	return nil

}

// Transaction Addresses

func GetTxAddress(txId uint64, addrId uint64) *model.TransactionAddress {
	//defer util.TimeTrack(time.Now(), "GetTxAddress")
	if model.TransactionAddressExistsGP(txId, addrId) {
		txAddress, err := model.FindTransactionAddressG(txId, addrId)
		if err != nil {
			logrus.Error("Datastore(GETTXADDRESS): ", err)
		}
		return txAddress
	}
	return nil
}

func PutTxAddress(txAddress *model.TransactionAddress) error {
	//defer util.TimeTrack(time.Now(), "PutTxAddres")
	if txAddress != nil {

		var err error
		if model.TransactionAddressExistsGP(txAddress.TransactionID, txAddress.AddressID) {
			err = txAddress.UpdateG()
		} else {
			err = txAddress.InsertG()

		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTTXADDRESS): ", err)
			return err
		}
	}

	return nil
}

//Claims

func GetClaim(addr string) *model.Claim {
	claimIdMatch := qm.Where(model.ClaimColumns.ClaimID+"=?", addr)

	if model.ClaimsG(claimIdMatch).ExistsP() {

		claim, err := model.ClaimsG(claimIdMatch).One()
		if err != nil {
			logrus.Error("Datastore(GETCLAIM): ", err)
		}
		return claim
	}

	return nil
}

func PutClaim(claim *model.Claim) error {

	if claim != nil {

		var err error
		if model.ClaimExistsGP(claim.ID) {
			err = claim.UpdateG()
		} else {
			err = claim.InsertG()
		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTCLAIM): ", err)
			return err
		}
	}
	return nil
}

//Supports

func GetSupport(txHash string, vout uint) *model.Support {
	txHashMatch := qm.Where(model.SupportColumns.TransactionHash+"=?", txHash)
	voutMatch := qm.Where(model.SupportColumns.Vout+"=?", vout)

	if model.SupportsG(txHashMatch, voutMatch).ExistsP() {

		support, err := model.SupportsG(txHashMatch, voutMatch).One()
		if err != nil {
			logrus.Error("Datastore(GETSUPPORT): ", err)
		}
		return support
	}
	return nil
}

func PutSupport(support *model.Support) error {

	if support != nil {

		var err error
		if model.ClaimExistsGP(support.ID) {
			err = support.UpdateG()
		} else {
			err = support.InsertG()
		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTSUPPORT): ", err)
			return err
		}
	}
	return nil
}
