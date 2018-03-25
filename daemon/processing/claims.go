package processing

import (
	"encoding/hex"
	"strconv"

	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/errors"
	"github.com/lbryio/lbry.go/util"
	"github.com/lbryio/lbryschema.go/pb"

	"github.com/sirupsen/logrus"
)

func processAsClaim(script []byte, vout model.Output, tx model.Transaction) (address *string, err error) {
	var pubkeyscript []byte
	var name string
	var claimid string
	if lbrycrd.IsClaimNameScript(script) {
		name, claimid, pubkeyscript, err = processClaimNameScript(&script, vout, tx)
		if err != nil {
			return nil, err
		}
		return nil, nil
	} else if lbrycrd.IsClaimSupportScript(script) {
		name, claimid, pubkeyscript, err = processClaimSupportScript(&script, vout, tx)
		if err != nil {
			return nil, err
		}
		return nil, nil
	} else if lbrycrd.IsClaimUpdateScript(script) {
		name, claimid, pubkeyscript, err = processClaimUpdateScript(&script, vout, tx)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
	pksAddress := lbrycrd.GetAddressFromPublicKeyScript(pubkeyscript)
	address = &pksAddress
	logrus.Debug("Handled Claim: ", " Name ", name, ", ClaimID ", claimid)

	return nil, errors.Base("Not a claim -- " + hex.EncodeToString(script))
}

func processClaimNameScript(script *[]byte, vout model.Output, tx model.Transaction) (name string, claimid string, pkscript []byte, err error) {
	name, value, pubkeyscript, err := lbrycrd.ParseClaimNameScript(*script)
	if err != nil {
		errors.Prefix("Claim name processing error: ", err)
		return name, claimid, pubkeyscript, err
	}
	pbClaim, err := lbrycrd.DecodeClaimValue(name, value)
	if pbClaim != nil {
		claimid, err := util.ClaimIDFromOutpoint(vout.TransactionHash, int(vout.Vout))
		if err != nil {
			return name, "", pubkeyscript, err
		}
		claim, err := processClaim(pbClaim, vout, tx)
		if err != nil {
			return name, claimid, pubkeyscript, err
		}
		claim.ClaimID = claimid
		claim.Name = name //
		//claim.TransactionTime.Uint = tx.TransactionTime //ToDo uncomment when DB rebuilt
		claim.InsertG()
	}
	if err != nil {
		logrus.Error(err)
		return name, claimid, pubkeyscript, nil
	}

	return name, claimid, pubkeyscript, err
}

func processClaimSupportScript(script *[]byte, vout model.Output, tx model.Transaction) (name string, claimid string, pubkeyscript []byte, err error) {
	name, claimid, pubkeyscript, err = lbrycrd.ParseClaimSupportScript(*script)
	if err != nil {
		errors.Prefix("Claim support processing error: ", err)
		return name, claimid, pubkeyscript, err
	}
	//log.Debug("ClaimSupport ", name, " ClaimId ", claimid)

	return name, claimid, pubkeyscript, err
}

func processClaimUpdateScript(script *[]byte, vout model.Output, tx model.Transaction) (name string, claimId string, pubkeyscript []byte, err error) {
	name, claimId, value, pubkeyscript, err := lbrycrd.ParseClaimUpdateScript(*script)
	if err != nil {
		errors.Prefix("Claim update processing error: ", err)
		return name, claimId, pubkeyscript, err
	}
	claim, err := lbrycrd.DecodeClaimValue(name, value)
	if claim != nil {
		//logrus.Debug("ClaimUpdate ", name, " ClaimId ", claimId)
	}
	return name, claimId, pubkeyscript, err
}

func processClaim(pbClaim *pb.Claim, output model.Output, tx model.Transaction) (model.Claim, error) {
	claim := model.Claim{}
	claim.SetTransactionByHashG(false, &tx)
	claim.Vout = output.Vout
	claim.Version = pbClaim.GetVersion().String()
	setSourceInfo(claim, pbClaim)
	setMetaDataInfo(claim, pbClaim)
	setCertificateInfo(claim, pbClaim)

	return claim, nil
}

func setCertificateInfo(claim model.Claim, pbClaim *pb.Claim) {

}

func setMetaDataInfo(claim model.Claim, pbClaim *pb.Claim) {
	stream := pbClaim.GetStream()
	if stream != nil {
		metadata := stream.GetMetadata()
		if metadata != nil {
			claim.Title.String = metadata.GetTitle()
			claim.Title.Valid = true

			claim.Description.String = metadata.GetDescription()
			claim.Description.Valid = true

			claim.Language.String = metadata.GetLanguage().String()
			claim.Language.Valid = true

			claim.Author.String = metadata.GetAuthor()
			claim.Author.Valid = true

			claim.ThumbnailURL.String = metadata.GetThumbnail()
			claim.ThumbnailURL.Valid = true

			fee := metadata.GetFee()
			if fee != nil {
				claim.FeeCurrency.String = fee.GetCurrency().String()
				claim.FeeCurrency.Valid = true

				claim.Fee = strconv.FormatFloat(float64(fee.GetAmount()), 'f', -1, 32)
			}
		}
	}
}

func setSourceInfo(claim model.Claim, pbClaim *pb.Claim) {
	stream := pbClaim.GetStream()
	if stream != nil {
		source := stream.GetSource()
		if source != nil {
			contentType := source.GetContentType()
			if contentType != "" {
				claim.ContentType.String = contentType
				claim.ContentType.Valid = true
			}
		}
	}
}
