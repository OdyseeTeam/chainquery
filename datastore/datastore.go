package datastore

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

// GetOutput makes creating,retrieving,updating the model type simplified.
func GetOutput(txHash string, vout uint) *model.Output {
	defer util.TimeTrack(time.Now(), "GetOutput", "mysqlprofile")
	txHashMatch := qm.Where(model.OutputColumns.TransactionHash+"=?", txHash)
	vOutMatch := qm.And(model.OutputColumns.Vout+"=?", vout)
	output, err := model.Outputs(txHashMatch, vOutMatch).OneG()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		logrus.Error("Datastore(GETOUTPUT): ", err)
	}
	return output
}

// PutOutput makes creating,retrieving,updating the model type simplified.
func PutOutput(output *model.Output, columns boil.Columns) error {
	defer util.TimeTrack(time.Now(), "PutOutput", "mysqlprofile")
	txHashMatch := qm.Where(model.OutputColumns.TransactionHash+"=?", output.TransactionHash)
	vOutMatch := qm.And(model.OutputColumns.Vout+"=?", output.Vout)
	existingOutput, err := model.Outputs(txHashMatch, vOutMatch).OneG()
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errors.Prefix("Datastore(PUTOUTPUT)", err)
	}
	if existingOutput != nil {
		output.ID = existingOutput.ID
		err = output.UpdateG(columns)
	} else {
		err = output.InsertG(boil.Infer())
	}
	if err != nil {
		return errors.Prefix("Datastore(PUTOUTPUT)", err)
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
	input, err := model.Inputs(txHashMatch, txCoinBaseMatch, prevHashMatch, prevNMatch).OneG()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		logrus.Error("Datastore(GETINPUT): ", err)
	}
	return input
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
			return errors.Prefix("Datastore(PUTINPUT)", err)
		}
		if exists {
			input.Modified = time.Now()
			err = input.UpdateG(boil.Infer())
		} else {
			err = input.InsertG(boil.Infer())

		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTINPUT)", err)
			return err
		}
	}

	return nil
}

// GetAddress makes creating,retrieving,updating the model type simplified.
func GetAddress(addr string) *model.Address {
	defer util.TimeTrack(time.Now(), "GetAddress", "mysqlprofile")
	addrMatch := qm.Where(model.AddressColumns.Address+"=?", addr)

	address, err := model.Addresses(addrMatch).OneG()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		logrus.Warning("Datastore(GETADDRESS): ", err)
	}
	return address
}

//PutAddress  makes creating,retrieving,updating the model type simplified.
func PutAddress(address *model.Address) error {
	defer util.TimeTrack(time.Now(), "PutAddress", "mysqlprofile")
	if address != nil {

		var err error
		exists, err := model.AddressExistsG(address.ID)
		if err != nil {
			return errors.Prefix("Datastore(PUTADDRESS)", err)
		}
		if exists {
			address.ModifiedAt = time.Now()
			err = address.UpdateG(boil.Infer())
		} else {
			err = address.InsertG(boil.Infer())

		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTADDRESS)", err)
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

// UpdateTxAddressAmounts updates the credit and debit amounts
func UpdateTxAddressAmounts(txAddress *model.TransactionAddress) error {
	defer util.TimeTrack(time.Now(), "UpdateTxAddressAmounts", "mysqlprofile")
	err := txAddress.UpdateG(boil.Infer())
	if err != nil {
		return errors.Prefix("Datastore(PUTTXADDRESS)", err)
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
			return errors.Prefix("Datastore(PUTTXADDRESS)", err)
		}
		if exists {
			err = txAddress.UpdateG(boil.Infer())
		} else {
			err = txAddress.InsertG(boil.Infer())
		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTTXADDRESS)", err)
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
			return errors.Prefix("Datastore(PUTCLAIM)", err)
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
			err = errors.Prefix("Datastore(PUTCLAIM)", err)
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
	support, err := model.Supports(txHashMatch, voutMatch).OneG()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		logrus.Warning("Datastore(GETSUPPORT): ", err)
	}
	return support
}

// PutSupport makes creating,retrieving,updating the model type simplified.
func PutSupport(support *model.Support) error {
	defer util.TimeTrack(time.Now(), "PutSupport", "mysqlprofile")
	err := support.UpsertG(boil.Infer(), boil.Infer())
	if err != nil {
		return errors.Prefix("Datastore(PUTSUPPORT)", err)
	}
	return nil
}

// GetTag makes creating,retrieving,updating the model type simplified.
func GetTag(tagName string) *model.Tag {
	defer util.TimeTrack(time.Now(), "GetTag", "mysqlprofile")
	tagMatch := qm.Where(model.TagColumns.Tag+"=?", tagName)
	tag, err := model.Tags(tagMatch).OneG()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		logrus.Warningf("Datastore(GETTAG): %s", err.Error())
	}
	return tag
}

// PutTag makes creating,retrieving,updating the model type simplified.
func PutTag(tag *model.Tag) error {
	defer util.TimeTrack(time.Now(), "PutTag", "mysqlprofile")
	err := tag.UpsertG(boil.None(), boil.Infer())
	if err != nil {
		err = errors.Prefix("Datastore(PUTTAG)", err)
		return err
	}
	return nil
}

// GetClaimTag makes creating,retrieving,updating the model type simplified.
func GetClaimTag(tagID uint64, claimID string) *model.ClaimTag {
	defer util.TimeTrack(time.Now(), "GetClaimTag", "mysqlprofile")
	claimIDMatch := model.ClaimTagWhere.ClaimID.EQ(claimID)
	tagIDMatch := model.ClaimTagWhere.TagID.EQ(null.Uint64From(tagID))

	claimTag, err := model.ClaimTags(tagIDMatch, claimIDMatch).OneG()
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		logrus.Warningf("Datastore(GETCLAIMTAG): %s", err.Error())
	}
	return claimTag
}

// PutClaimTag makes creating,retrieving,updating the model type simplified.
func PutClaimTag(claimTag *model.ClaimTag) error {
	defer util.TimeTrack(time.Now(), "PutClaimTag", "mysqlprofile")
	//using UpsertG fails because sqlboiler doesn't consider multi column unique keys as valid. hence the manual "upsert" logic here.
	query := fmt.Sprintf(`INSERT INTO claim_tag (%s, %s) VALUES(?, ?) ON DUPLICATE KEY UPDATE id=id`, model.ClaimTagColumns.TagID, model.ClaimTagColumns.ClaimID)
	_, err := boil.GetDB().Exec(query, claimTag.TagID.Uint64, claimTag.ClaimID)
	if err != nil {
		return errors.Prefix("Datastore(PUTCLAIMTAG)", err)
	}
	return nil
}

// GetPurchase makes creating,retrieving,updating the model type simplified.
func GetPurchase(txHash string, vout uint, claimID string) *model.Purchase {
	defer util.TimeTrack(time.Now(), "GetPurchase", "mysqlprofile")
	claimIDMatch := model.PurchaseWhere.ClaimID.EQ(null.StringFrom(claimID))
	txHashMatch := model.PurchaseWhere.TransactionByHashID.EQ(null.StringFrom(txHash))
	voutMatch := model.PurchaseWhere.Vout.EQ(vout)

	exists, err := model.Purchases(claimIDMatch, txHashMatch, voutMatch).ExistsG()
	if err != nil {
		logrus.Warning("Datastore(GETPURCHASE): ", err)
	}
	if exists {
		purchase, err := model.Purchases(claimIDMatch, txHashMatch, voutMatch).OneG()
		if err != nil {
			logrus.Warning("Datastore(GETPURCHASE): ", err)
		}
		return purchase
	}
	return nil
}

// PutPurchase makes creating,retrieving,updating the model type simplified.
func PutPurchase(purchase *model.Purchase) error {
	defer util.TimeTrack(time.Now(), "PutPurchase", "mysqlprofile")
	if purchase != nil {

		var err error
		claimIDMatch := model.PurchaseWhere.ClaimID.EQ(null.StringFrom(purchase.ClaimID.String))
		txHashMatch := model.PurchaseWhere.TransactionByHashID.EQ(null.StringFrom(purchase.TransactionByHashID.String))
		voutMatch := model.PurchaseWhere.Vout.EQ(purchase.Vout)
		exists, err := model.Purchases(claimIDMatch, txHashMatch, voutMatch).ExistsG()
		if err != nil {
			return errors.Prefix("Datastore(PUTPURCHASE)", err)
		}
		if exists {
			purchase.Modified = time.Now()
			err = purchase.UpdateG(boil.Infer())
		} else {
			err = purchase.InsertG(boil.Infer())
		}
		if err != nil {
			err = errors.Prefix("Datastore(PUTPURCHASE)", err)
			return err
		}
	}
	return nil
}
