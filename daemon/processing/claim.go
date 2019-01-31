package processing

import (
	"encoding/hex"
	"encoding/json"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/global"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/errors"
	util "github.com/lbryio/lbry.go/lbrycrd"
	"github.com/lbryio/lbryschema.go/address/base58"
	c "github.com/lbryio/lbryschema.go/claim"
	"github.com/lbryio/types/go"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
)

func processAsClaim(script []byte, vout model.Output, tx model.Transaction, blockHeight uint64) (address *string, claimID *string, err error) {
	var pubkeyscript []byte
	var name string
	var claimid string
	if lbrycrd.IsClaimNameScript(script) {
		name, claimid, pubkeyscript, err = processClaimNameScript(&script, vout, tx, blockHeight)
		if err != nil {
			return nil, nil, err
		}
	} else if lbrycrd.IsClaimSupportScript(script) {
		name, claimid, pubkeyscript, err = processClaimSupportScript(&script, vout, tx)
		if err != nil {
			return nil, nil, err
		}
	} else if lbrycrd.IsClaimUpdateScript(script) {
		name, claimid, pubkeyscript, err = processClaimUpdateScript(&script, vout, tx, blockHeight)
		if err != nil {
			return nil, nil, err
		}
	} else {
		return nil, nil, errors.Base("Not a claim -- " + hex.EncodeToString(script))
	}
	pksAddress := lbrycrd.GetAddressFromPublicKeyScript(pubkeyscript)
	address = &pksAddress
	logrus.Debug("Handled Claim: ", " Name ", name, ", ClaimID ", claimid)

	return address, &claimid, nil
}

func processClaimNameScript(script *[]byte, vout model.Output, tx model.Transaction, blockHeight uint64) (name string, claimid string, pkscript []byte, err error) {
	claimid, err = util.ClaimIDFromOutpoint(vout.TransactionHash, int(vout.Vout))
	if err != nil {
		return name, "", pkscript, err
	}
	name, value, pkscript, err := lbrycrd.ParseClaimNameScript(*script)
	if err != nil {
		err := errors.Prefix("Claim name script parsing error: ", err)
		return name, claimid, pkscript, err
	}
	helper, err := c.DecodeClaimBytes(value, global.BlockChainName)
	if err != nil {
		logrus.Debug("saving non-conforming claim - Name: ", name, " ClaimID: ", claimid)
		saveUnknownClaim(name, claimid, false, value, vout, tx)
		return name, claimid, pkscript, nil
	}
	if helper.Claim == nil {
		err := errors.Base("Produced null pbClaim-> " + name + " " + claimid)
		return name, claimid, pkscript, err
	}
	claim := datastore.GetClaim(claimid)
	claim, err = processClaim(helper.Claim, claim, value, vout, tx)
	if err != nil {
		return name, claimid, pkscript, err
	}
	claim.ClaimID = claimid
	claim.Name = name
	claim.TransactionTime = tx.TransactionTime
	claim.ClaimAddress = lbrycrd.GetAddressFromPublicKeyScript(pkscript)
	claim.Height = uint(blockHeight)
	err = datastore.PutClaim(claim)

	return name, claimid, pkscript, err
}

func processClaimSupportScript(script *[]byte, vout model.Output, tx model.Transaction) (name string, claimid string, pubkeyscript []byte, err error) {
	name, claimid, pubkeyscript, err = lbrycrd.ParseClaimSupportScript(*script)
	if err != nil {
		err := errors.Prefix("Claim support processing error: ", err)
		return name, claimid, pubkeyscript, err
	}
	support := datastore.GetSupport(tx.Hash, vout.Vout)
	support = processSupport(claimid, support, vout, tx)
	if err := datastore.PutSupport(support); err != nil {
		logrus.Debug("Support for unknown claim! ", claimid)
	}

	return name, claimid, pubkeyscript, err
}

func processClaimUpdateScript(script *[]byte, vout model.Output, tx model.Transaction, blockHeight uint64) (name string, claimID string, pubkeyscript []byte, err error) {
	name, claimID, value, pubkeyscript, err := lbrycrd.ParseClaimUpdateScript(*script)
	if err != nil {
		err := errors.Prefix("Claim update processing error: ", err)
		return name, claimID, pubkeyscript, err
	}
	helper, err := c.DecodeClaimBytes(value, global.BlockChainName)
	if err != nil {
		logrus.Debug("saving non-conforming claim - Update: ", name, " ClaimID: ", claimID)
		saveUnknownClaim(name, claimID, true, value, vout, tx)
		return name, claimID, pubkeyscript, nil
	}
	if helper.Claim != nil && err == nil {
		claim := datastore.GetClaim(claimID)
		claim, err := processUpdateClaim(helper.Claim, claim, value)
		if err != nil {
			return name, claimID, pubkeyscript, err
		}
		if claim == nil {
			logrus.Debug("ClaimUpdate for non-existent claim! ", claimID, " ", tx.Hash, " ", vout.Vout)
			return name, claimID, pubkeyscript, err
		}
		claim.TransactionTime = tx.TransactionTime
		claim.ClaimAddress = lbrycrd.GetAddressFromPublicKeyScript(pubkeyscript)
		claim.Height = uint(blockHeight)
		claim.TransactionHashID.SetValid(tx.Hash)
		claim.Vout = vout.Vout
		if err := datastore.PutClaim(claim); err != nil {
			logrus.Debug("Claim updates to invalid certificate claim. ", claim.PublisherID)
			if logrus.GetLevel() == logrus.DebugLevel {
				logrus.WithError(err)
			}
		}
	}
	return name, claimID, pubkeyscript, err
}

func processClaim(pbClaim *pb.Claim, claim *model.Claim, value []byte, output model.Output, tx model.Transaction) (*model.Claim, error) {
	if claim == nil {
		claim = &model.Claim{}
	}
	claim.TransactionHashID.SetValid(tx.Hash)
	claim.Vout = output.Vout
	claim.Version = pbClaim.GetVersion().String()
	claim.ValueAsHex = hex.EncodeToString(value)
	claim.ClaimType = int8(pb.Claim_ClaimType_value[pbClaim.GetClaimType().String()])

	// pbClaim JSON
	if claimHelper, err := c.DecodeClaimHex(claim.ValueAsHex, global.BlockChainName); err == nil {
		if jsonvalue, err := claimHelper.RenderJSON(); err == nil {
			claim.ValueAsJSON.SetValid(jsonvalue)
		}
	}

	setSourceInfo(claim, pbClaim)
	setMetaDataInfo(claim, pbClaim)
	setPublisherInfo(claim, pbClaim)
	setCertificateInfo(claim, pbClaim)

	return claim, nil
}

func processSupport(claimID string, support *model.Support, output model.Output, tx model.Transaction) *model.Support {
	if support == nil {
		support = &model.Support{}
	}

	support.TransactionHashID.SetValid(tx.Hash)
	support.Vout = output.Vout
	support.SupportAmount = output.Value.Float64
	if claim := datastore.GetClaim(claimID); claim != nil {
		support.SupportedClaimID = claimID
		return support
	}
	logrus.Debug("Claim Support for claim ", claimID, " is a non-existent claim.")
	return support

}

func processUpdateClaim(pbClaim *pb.Claim, claim *model.Claim, value []byte) (*model.Claim, error) {
	if claim == nil {
		return nil, nil
	}
	claim.Version = pbClaim.GetVersion().String()
	claim.ValueAsHex = hex.EncodeToString(value)

	// pbClaim JSON
	if claimHelper, err := c.DecodeClaimHex(claim.ValueAsHex, "lbrycrd_main"); err == nil {
		if jsonvalue, err := claimHelper.RenderJSON(); err == nil {
			claim.ValueAsJSON.SetValid(jsonvalue)
		}
	}

	setSourceInfo(claim, pbClaim)
	setMetaDataInfo(claim, pbClaim)
	setPublisherInfo(claim, pbClaim)
	setCertificateInfo(claim, pbClaim)

	return claim, nil
}

func setPublisherInfo(claim *model.Claim, pbClaim *pb.Claim) {
	claim.IsCertProcessed = true
	claim.IsCertValid = false
	claim.PublisherID = null.NewString("", false)
	claim.PublisherSig = null.NewString("", false)
	if pbClaim.GetPublisherSignature() != nil {
		claim.IsCertProcessed = false
		publisherClaimID := hex.EncodeToString(pbClaim.GetPublisherSignature().GetCertificateId())
		claim.PublisherID.SetValid(publisherClaimID)
		claim.PublisherSig.SetValid(hex.EncodeToString(pbClaim.GetPublisherSignature().GetSignature()))
	}
}

func setCertificateInfo(claim *model.Claim, pbClaim *pb.Claim) {
	claim.IsCertProcessed = false
	claim.Certificate = null.NewString("", false)
	if pbClaim.GetClaimType() == pb.Claim_certificateType {
		claim.IsCertProcessed = true
		certificate := pbClaim.GetCertificate()
		certBytes, err := json.Marshal(certificate)
		if err != nil {
			logrus.Error("Could not form json from certificate")
		}
		claim.Certificate.SetValid(string(certBytes))
	}
}

func setMetaDataInfo(claim *model.Claim, pbClaim *pb.Claim) {
	resetMetadata(claim)
	stream := pbClaim.GetStream()
	if stream != nil {
		metadata := stream.GetMetadata()
		if metadata != nil {
			claim.Title.SetValid(metadata.GetTitle())
			claim.Description.SetValid(metadata.GetDescription())
			claim.Language.SetValid(metadata.GetLanguage().String())
			claim.Author.SetValid(metadata.GetAuthor())
			claim.ThumbnailURL.SetValid(metadata.GetThumbnail())
			claim.IsNSFW = metadata.GetNsfw()
			if metadata.License != nil {
				claim.License.SetValid(metadata.GetLicense())
			}
			if metadata.LicenseUrl != nil {
				claim.LicenseURL.SetValid(metadata.GetLicenseUrl())
			}
			if metadata.Preview != nil {
				claim.Preview.SetValid(metadata.GetPreview())
			}

			fee := metadata.GetFee()
			if fee != nil {
				claim.FeeCurrency.SetValid(fee.GetCurrency().String())

				claim.Fee = float64(fee.GetAmount())
				claim.FeeAddress = base58.EncodeBase58(fee.GetAddress())
			}
		}
	}
}

func resetMetadata(claim *model.Claim) {
	claim.Title = null.NewString("", false)
	claim.Description = null.NewString("", false)
	claim.Language = null.NewString("", false)
	claim.Author = null.NewString("", false)
	claim.ThumbnailURL = null.NewString("", false)
	claim.IsNSFW = false
	claim.FeeCurrency = null.NewString("", false)
	claim.Fee = 0.0
	claim.FeeAddress = ""
	claim.License = null.NewString("", false)
	claim.LicenseURL = null.NewString("", false)
	claim.Preview = null.NewString("", false)
}

func setSourceInfo(claim *model.Claim, pbClaim *pb.Claim) {
	claim.ContentType = null.NewString("", false)
	claim.SDHash = null.NewString("", false)
	stream := pbClaim.GetStream()
	if stream != nil {
		source := stream.GetSource()
		if source != nil {
			contentType := source.GetContentType()
			if contentType != "" {
				claim.ContentType.SetValid(contentType)
				if source.GetSourceType() == pb.Source_lbry_sd_hash {
					claim.SDHash.SetValid(hex.EncodeToString(source.GetSource()))
				}
			}
		}
	}
}

func saveUnknownClaim(name string, claimid string, isUpdate bool, value []byte, vout model.Output, tx model.Transaction) {
	abnormalClaim := model.AbnormalClaim{}
	abnormalClaim.Vout = vout.Vout
	abnormalClaim.Name = name
	abnormalClaim.ClaimID = claimid
	abnormalClaim.IsUpdate = isUpdate
	abnormalClaim.TransactionHash.SetValid(vout.TransactionHash)
	abnormalClaim.ValueAsHex = hex.EncodeToString(value)
	abnormalClaim.BlockHash = tx.BlockHashID

	var js map[string]interface{} //JSON Map
	if json.Unmarshal(value, &js) == nil {
		abnormalClaim.ValueAsJSON.SetValid(string(value))
	}

	abnormalClaim.OutputID = vout.ID
	if err := abnormalClaim.InsertG(boil.Infer()); err != nil {
		logrus.Error("UnknownClaim Saving Error: ", err)
	}

}
