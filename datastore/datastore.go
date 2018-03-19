package datastore

import (
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/errors.go"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

//Outputs
func GetOutput(txHash string, vout uint) *model.Output {

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

	if address != nil {

		var err error
		if model.AddressExistsGP(address.ID) {
			err = address.UpdateG(
				// Needed to avoid saving Balance Column which is calculated
				model.AddressColumns.Address,
				model.AddressColumns.FirstSeen,
				model.AddressColumns.TagURL,
				model.AddressColumns.Tag,
				model.AddressColumns.TotalReceived,
				model.AddressColumns.TotalSent,
			)
		} else {
			err = address.InsertG(
				// Needed to avoid saving Balance Column which is calculated
				model.AddressColumns.Address,
				model.AddressColumns.FirstSeen,
				model.AddressColumns.TagURL,
				model.AddressColumns.Tag,
				model.AddressColumns.TotalReceived,
				model.AddressColumns.TotalSent,
			)

		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTADDRESS): ", err)
			return err
		}
	}

	return nil

}

func PutTxAddress(txAddress *model.TransactionAddress) error {

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
