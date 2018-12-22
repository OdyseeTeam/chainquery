package datastore

import (
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"

	"time"

	"github.com/lbryio/chainquery/util"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

// GetOutput makes creating,retrieving,updating the model type simplified.
func GetOutput(txHash string, vout uint) *model.Output {
	defer util.TimeTrack(time.Now(), "GetOutput", "mysqlprofile")
	txHashMatch := qm.Where(model.OutputColumns.TransactionHash+"=?", txHash)
	vOutMatch := qm.And(model.OutputColumns.Vout+"=?", vout)
	key := outputKey{txHash: txHash, vout: vout}

	if checkOutputCache(key) || model.Outputs(txHashMatch, vOutMatch).ExistsGP() {
		output, err := model.Outputs(txHashMatch, vOutMatch).OneG()
		if err != nil {
			logrus.Error("Datastore(GETOUTPUT): ", err)
		}
		return output
	}

	return nil
}

// PutOutput makes creating,retrieving,updating the model type simplified.
func PutOutput(output *model.Output, whitelist ...string) error {
	defer util.TimeTrack(time.Now(), "PutOutput", "mysqlprofile")
	if output != nil {
		txHashMatch := qm.Where(model.OutputColumns.TransactionHash+"=?", output.TransactionHash)
		vOutMatch := qm.And(model.OutputColumns.Vout+"=?", output.Vout)
		var err error
		if model.Outputs(txHashMatch, vOutMatch).ExistsGP() {
			output.ModifiedAt = time.Now()
			err = output.UpdateG(boil.Whitelist(whitelist...))
		} else {
			err = output.InsertG(boil.Infer())
			if err != nil {
				output.ModifiedAt = time.Now()
				err = output.UpdateG(boil.Whitelist(whitelist...))
			}
		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTOUTPUT): ", err)
			return err
		}
	}

	return nil
}

// GetInput makes creating,retrieving,updating the model type simplified.
func GetInput(txHash string, isCoinBase bool, prevHash string, prevN uint) *model.Input {
	defer util.TimeTrack(time.Now(), "GetInput", "mysqlprofile")
	//Unique
	txHashMatch := qm.Where(model.InputColumns.TransactionHash+"=?", txHash)
	txCoinBaseMatch := qm.Where(model.InputColumns.IsCoinbase+"=?", isCoinBase)
	prevHashMatch := qm.Where(model.InputColumns.PrevoutHash+"=?", prevHash)
	prevNMatch := qm.And(model.InputColumns.PrevoutN+"=?", prevN)

	if model.Inputs(txHashMatch, txCoinBaseMatch, prevHashMatch, prevNMatch).ExistsGP() {
		input, err := model.Inputs(txHashMatch, txCoinBaseMatch, prevHashMatch, prevNMatch).OneG()
		if err != nil {
			logrus.Error("Datastore(GETINPUT): ", err)
		}
		return input
	}

	return nil
}

//PutInput makes creating,retrieving,updating the model type simplified.
func PutInput(input *model.Input) error {
	defer util.TimeTrack(time.Now(), "PutInput", "mysqlprofile")
	if input != nil {
		//Unique
		txHashMatch := qm.Where(model.InputColumns.TransactionHash+"=?", input.TransactionHash)
		txCoinBaseMatch := qm.Where(model.InputColumns.IsCoinbase+"=?", input.IsCoinbase)
		prevHashMatch := qm.Where(model.InputColumns.PrevoutHash+"=?", input.PrevoutHash)
		prevNMatch := qm.And(model.InputColumns.PrevoutN+"=?", input.PrevoutN)

		var err error
		if model.Inputs(txHashMatch, txCoinBaseMatch, prevHashMatch, prevNMatch).ExistsGP() {
			input.Modified = time.Now()
			err = input.UpdateG(boil.Infer())
		} else {
			err = input.InsertG(boil.Infer())

		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTINPUT): ", err)
			return err
		}
	}

	return nil
}

// GetAddress makes creating,retrieving,updating the model type simplified.
func GetAddress(addr string) *model.Address {
	defer util.TimeTrack(time.Now(), "GetAddress", "mysqlprofile")
	addrMatch := qm.Where(model.AddressColumns.Address+"=?", addr)

	if model.Addresses(addrMatch).ExistsGP() {

		address, err := model.Addresses(addrMatch).OneG()
		if err != nil {
			logrus.Error("Datastore(GETADDRESS): ", err)
		}
		return address
	}

	return nil
}

//PutAddress  makes creating,retrieving,updating the model type simplified.
func PutAddress(address *model.Address) error {
	defer util.TimeTrack(time.Now(), "PutAddress", "mysqlprofile")
	if address != nil {

		var err error
		if model.AddressExistsGP(address.ID) {
			address.ModifiedAt = time.Now()
			err = address.UpdateG(boil.Infer())
		} else {
			err = address.InsertG(boil.Infer())

		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTADDRESS): ", err)
			return err
		}
	}

	return nil

}

// GetTxAddress makes creating,retrieving,updating the model type simplified.
func GetTxAddress(txID uint64, addrID uint64) *model.TransactionAddress {
	defer util.TimeTrack(time.Now(), "GetTxAddress", "mysqlprofile")
	key := txAddressKey{txID: txID, addrID: addrID}
	if checkTxAddrCache(key) || model.TransactionAddressExistsGP(txID, addrID) {

		txAddress, err := model.FindTransactionAddressG(txID, addrID)
		if err != nil {
			logrus.Error("Datastore(GETTXADDRESS): ", err)
		}
		return txAddress
	}
	return nil
}

// PutTxAddress makes creating,retrieving,updating the model type simplified.
func PutTxAddress(txAddress *model.TransactionAddress) error {
	defer util.TimeTrack(time.Now(), "PutTxAddres", "mysqlprofile")
	if txAddress != nil {
		key := txAddressKey{txID: txAddress.TransactionID, addrID: txAddress.AddressID}
		var err error
		if checkTxAddrCache(key) || model.TransactionAddressExistsGP(txAddress.TransactionID, txAddress.AddressID) {
			err = txAddress.UpdateG(boil.Infer())
		} else {
			err = txAddress.InsertG(boil.Infer())
			addToTxAddrCache(key)
		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTTXADDRESS): ", err)
			return err
		}
	}

	return nil
}

// GetClaim makes creating,retrieving,updating the model type simplified.
func GetClaim(addr string) *model.Claim {
	defer util.TimeTrack(time.Now(), "GetClaim", "mysqlprofile")
	claimIDMatch := qm.Where(model.ClaimColumns.ClaimID+"=?", addr)

	if model.Claims(claimIDMatch).ExistsGP() {

		claim, err := model.Claims(claimIDMatch).OneG()
		if err != nil {
			logrus.Error("Datastore(GETCLAIM): ", err)
		}
		return claim
	}

	return nil
}

// PutClaim makes creating,retrieving,updating the model type simplified.
func PutClaim(claim *model.Claim) error {
	defer util.TimeTrack(time.Now(), "PutClaim", "mysqlprofile")
	if claim != nil {

		var err error
		if model.ClaimExistsGP(claim.ID) {
			claim.ModifiedAt = time.Now()
			err = claim.UpdateG(boil.Infer())
		} else {
			err = claim.InsertG(boil.Infer())
			if err != nil {
				claim.ModifiedAt = time.Now()
				err = claim.UpdateG(boil.Infer())
			}
		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTCLAIM): ", err)
			return err
		}
	}
	return nil
}

// GetSupport makes creating,retrieving,updating the model type simplified.
func GetSupport(txHash string, vout uint) *model.Support {
	defer util.TimeTrack(time.Now(), "GetSupport", "mysqlprofile")
	txHashMatch := qm.Where(model.SupportColumns.TransactionHashID+"=?", txHash)
	voutMatch := qm.Where(model.SupportColumns.Vout+"=?", vout)

	if model.Supports(txHashMatch, voutMatch).ExistsGP() {

		support, err := model.Supports(txHashMatch, voutMatch).OneG()
		if err != nil {
			logrus.Error("Datastore(GETSUPPORT): ", err)
		}
		return support
	}
	return nil
}

// PutSupport makes creating,retrieving,updating the model type simplified.
func PutSupport(support *model.Support) error {
	defer util.TimeTrack(time.Now(), "PutSupport", "mysqlprofile")
	if support != nil {

		var err error
		if model.ClaimExistsGP(support.ID) {
			support.ModifiedAt = time.Now()
			err = support.UpdateG(boil.Infer())
		} else {
			err = support.InsertG(boil.Infer())
		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTSUPPORT): ", err)
			return err
		}
	}
	return nil
}
