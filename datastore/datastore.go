package datastore

import (
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/extras/errors"

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
	exists, err := model.Outputs(txHashMatch, vOutMatch).ExistsG()
	if err != nil {
		return nil
	}
	if exists {
		output, err := model.Outputs(txHashMatch, vOutMatch).OneG()
		if err != nil {
			logrus.Error("Datastore(GETOUTPUT): ", err)
		}
		return output
	}

	return nil
}

// PutOutput makes creating,retrieving,updating the model type simplified.
func PutOutput(output *model.Output, columns boil.Columns) error {
	defer util.TimeTrack(time.Now(), "PutOutput", "mysqlprofile")
	if output != nil {
		txHashMatch := qm.Where(model.OutputColumns.TransactionHash+"=?", output.TransactionHash)
		vOutMatch := qm.And(model.OutputColumns.Vout+"=?", output.Vout)
		var err error
		exists, err := model.Outputs(txHashMatch, vOutMatch).ExistsG()
		if err != nil {
			return errors.Prefix("Datastore(PUTOUTPUT): ", err)
		}
		if exists {
			output.ModifiedAt = time.Now()
			err = output.UpdateG(columns)
		} else {
			err = output.InsertG(boil.Infer())
			if err != nil {
				output.ModifiedAt = time.Now()
				err = output.UpdateG(columns)
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

	exists, err := model.Inputs(txHashMatch, txCoinBaseMatch, prevHashMatch, prevNMatch).ExistsG()
	if err != nil {
		logrus.Warning("Datastore(GETINPUT): ", err)
	}
	if exists {
		input, err := model.Inputs(txHashMatch, txCoinBaseMatch, prevHashMatch, prevNMatch).OneG()
		if err != nil {
			logrus.Warning("Datastore(GETINPUT): ", err)
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
		if input.PrevoutHash.IsZero() {
			prevHashMatch = qm.Where(model.InputColumns.PrevoutHash + " IS NULL")
		}
		prevNMatch := qm.And(model.InputColumns.PrevoutN+"=?", input.PrevoutN)
		if input.PrevoutN.IsZero() {
			prevNMatch = qm.And(model.InputColumns.PrevoutN + " IS NULL ")
		}

		var err error
		exists, err := model.Inputs(txHashMatch, txCoinBaseMatch, prevHashMatch, prevNMatch).ExistsG()
		if err != nil {
			return errors.Prefix("Datastore(PUTINPUT): ", err)
		}
		if exists {
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

	exists, err := model.Addresses(addrMatch).ExistsG()
	if err != nil {
		logrus.Warning("Datastore(GETADDRESS): ", err)
	}
	if exists {

		address, err := model.Addresses(addrMatch).OneG()
		if err != nil {
			logrus.Warning("Datastore(GETADDRESS): ", err)
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
		exists, err := model.AddressExistsG(address.ID)
		if err != nil {
			return errors.Prefix("Datastore(PUTADDRESS): ", err)
		}
		if exists {
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
	exists, err := model.TransactionAddressExistsG(txID, addrID)
	if err != nil {
		logrus.Warning("Datastore(GETTXADDRESS): ", err)
	}
	if exists {
		txAddress, err := model.FindTransactionAddressG(txID, addrID)
		if err != nil {
			logrus.Warning("Datastore(GETTXADDRESS): ", err)
		}
		return txAddress
	}
	return nil
}

// PutTxAddress makes creating,retrieving,updating the model type simplified.
func PutTxAddress(txAddress *model.TransactionAddress) error {
	defer util.TimeTrack(time.Now(), "PutTxAddres", "mysqlprofile")
	if txAddress != nil {
		var err error
		exists, err := model.TransactionAddressExistsG(txAddress.TransactionID, txAddress.AddressID)
		if err != nil {
			return errors.Prefix("Datastore(PUTTXADDRESS): ", err)
		}
		if exists {
			err = txAddress.UpdateG(boil.Infer())
		} else {
			err = txAddress.InsertG(boil.Infer())
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

	exists, err := model.Claims(claimIDMatch).ExistsG()
	if err != nil {
		logrus.Warning("Datastore(GETCLAIM): ", err)
	}
	if exists {

		claim, err := model.Claims(claimIDMatch).OneG()
		if err != nil {
			logrus.Warning("Datastore(GETCLAIM): ", err)
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
		exists, err := model.ClaimExistsG(claim.ID)
		if err != nil {
			return errors.Prefix("Datastore(PUTCLAIM): ", err)
		}
		if exists {
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

	exists, err := model.Supports(txHashMatch, voutMatch).ExistsG()
	if err != nil {
		logrus.Warning("Datastore(GETSUPPORT): ", err)
	}
	if exists {

		support, err := model.Supports(txHashMatch, voutMatch).OneG()
		if err != nil {
			logrus.Warning("Datastore(GETSUPPORT): ", err)
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
		exists, err := model.ClaimExistsG(support.ID)
		if err != nil {
			return errors.Prefix("Datastore(PUTSUPPORT): ", err)
		}
		if exists {
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

// GetTag makes creating,retrieving,updating the model type simplified.
func GetTag(tag string) *model.Tag {
	defer util.TimeTrack(time.Now(), "GetTag", "mysqlprofile")
	tagMatch := qm.Where(model.TagColumns.Tag+"=?", tag)

	exists, err := model.Tags(tagMatch).ExistsG()
	if err != nil {
		logrus.Warning("Datastore(GETTAG): ", err)
	}
	if exists {

		tag, err := model.Tags(tagMatch).OneG()
		if err != nil {
			logrus.Warning("Datastore(GETTAG): ", err)
		}
		return tag
	}
	return nil
}

// PutTag makes creating,retrieving,updating the model type simplified.
func PutTag(tag *model.Tag) error {
	defer util.TimeTrack(time.Now(), "PutTag", "mysqlprofile")
	if tag != nil {

		var err error
		exists, err := model.TagExistsG(tag.ID)
		if err != nil {
			return errors.Prefix("Datastore(PUTTAG): ", err)
		}
		if exists {
			tag.ModifiedAt = time.Now()
			err = tag.UpdateG(boil.Infer())
		} else {
			err = tag.InsertG(boil.Infer())
		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTTAG): ", err)
			return err
		}
	}
	return nil
}

// GetClaimTag makes creating,retrieving,updating the model type simplified.
func GetClaimTag(tagID uint64, claimID string) *model.ClaimTag {
	defer util.TimeTrack(time.Now(), "GetClaimTag", "mysqlprofile")
	tagIDMatch := qm.Where(model.ClaimTagColumns.TagID+"=?", tagID)
	claimIDMatch := qm.Where(model.ClaimTagColumns.ClaimID+"=?", claimID)

	exists, err := model.ClaimTags(tagIDMatch, claimIDMatch).ExistsG()
	if err != nil {
		logrus.Warning("Datastore(GETTAG): ", err)
	}
	if exists {

		claimTag, err := model.ClaimTags(tagIDMatch, claimIDMatch).OneG()
		if err != nil {
			logrus.Warning("Datastore(GETTAG): ", err)
		}
		return claimTag
	}
	return nil
}

// PutClaimTag makes creating,retrieving,updating the model type simplified.
func PutClaimTag(claimTag *model.ClaimTag) error {
	defer util.TimeTrack(time.Now(), "PutClaimTag", "mysqlprofile")
	if claimTag != nil {

		var err error
		exists, err := model.ClaimTagExistsG(claimTag.ID)
		if err != nil {
			return errors.Prefix("Datastore(PUTCLAIMTAG): ", err)
		}
		if exists {
			claimTag.ModifiedAt = time.Now()
			err = claimTag.UpdateG(boil.Infer())
		} else {
			err = claimTag.InsertG(boil.Infer())
		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTCLAIMTAG): ", err)
			return err
		}
	}
	return nil
}
