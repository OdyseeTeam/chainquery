package lbrycrd

import (
	"github.com/lbryio/lbryschema.go/pb"

	"encoding/hex"
	"encoding/json"
	"github.com/lbryio/chainquery/app/lbrycrd/schemas/schema_version_01"
	"github.com/lbryio/chainquery/app/lbrycrd/schemas/schema_version_02"
	"github.com/lbryio/chainquery/app/lbrycrd/schemas/schema_version_03"
	"github.com/lbryio/errors.go"
	"github.com/lbryio/lbryschema.go/claim"
)

//
func DecodeClaimValue(name string, value []byte) (*pb.Claim, error) {
	claim, err := decodeClaimFromValueBytes(value)
	if err != nil {
		v1Claim := new(schema_version_01.Claim)
		err := json.Unmarshal(value, v1Claim)
		if err != nil {
			v2Claim := new(schema_version_02.Claim)
			err := json.Unmarshal(value, v2Claim)
			if err != nil {
				v3Claim := new(schema_version_03.Claim)
				err := json.Unmarshal(value, v3Claim)
				if err != nil {
					return nil, errors.Base("Claim value has no matching verion - " + string(value))
				}
				claim, err = migrateV3Claim(*v3Claim)
				if err != nil {
					return nil, errors.Prefix("V3 Metadata Migration Error", err)
				}
			}
			claim, err = migrateV2Claim(*v2Claim)
			if err != nil {
				return nil, errors.Prefix("V2 Metadata Migration Error ", err)
			}
		}
		claim, err = migrateV1Claim(*v1Claim)
		if err != nil {
			return nil, errors.Prefix("V1 Metadata Migration Error ", err)
		}
	}

	return claim, nil
}

func decodeClaimFromValueBytes(value []byte) (*pb.Claim, error) {
	decoded, err := claim.DecodeClaimBytes(value, "lbrycrd_main")
	if err != nil {
		return nil, err
	}
	return decoded.Claim, nil
}

func newClaim() *pb.Claim {
	claim := new(pb.Claim)
	stream := new(pb.Stream)
	metadata := new(pb.Metadata)
	source := new(pb.Source)
	pubsig := new(pb.Signature)
	fee := new(pb.Fee)
	metadata.Fee = fee
	stream.Metadata = metadata
	stream.Source = source
	claim.Stream = stream
	claim.PublisherSignature = pubsig

	//Fee version
	feeVersion := pb.Fee__0_0_1
	claim.GetStream().GetMetadata().GetFee().Version = &feeVersion
	//Metadata version
	mdVersion := pb.Metadata__0_1_0
	claim.GetStream().GetMetadata().Version = &mdVersion
	//Source version
	srcVersion := pb.Source__0_0_1
	claim.GetStream().GetSource().Version = &srcVersion
	//Stream version
	strmVersion := pb.Stream__0_0_1
	claim.GetStream().Version = &strmVersion
	//Claim version
	clmVersion := pb.Claim__0_0_1
	claim.Version = &clmVersion
	//Claim type
	clmType := pb.Claim_streamType
	claim.ClaimType = &clmType

	return claim
}

func setMetaData(claim pb.Claim, author string, description string, language pb.Metadata_Language, license string,
	licenseURL *string, title string, thumbnail *string, nsfw bool) {

	claim.GetStream().GetMetadata().Author = &author
	claim.GetStream().GetMetadata().Description = &description
	claim.GetStream().GetMetadata().Language = &language
	claim.GetStream().GetMetadata().License = &license
	claim.GetStream().GetMetadata().Title = &title
	claim.GetStream().GetMetadata().Thumbnail = thumbnail
	claim.GetStream().GetMetadata().Nsfw = &nsfw
	claim.GetStream().GetMetadata().LicenseUrl = licenseURL

}

func migrateV1Claim(vClaim schema_version_01.Claim) (*pb.Claim, error) {

	claim := newClaim()
	//Stream
	// -->Universal
	setFee(vClaim.Fee, claim)
	// -->MetaData
	defaultNSFW := false
	language := pb.Metadata_Language(pb.Metadata_Language_value[vClaim.Language])
	setMetaData(*claim, vClaim.Author, vClaim.Description, language,
		vClaim.License, nil, vClaim.Title, vClaim.Thumbnail, defaultNSFW)
	// -->Source
	claim.GetStream().GetSource().ContentType = &vClaim.ContentType
	sourceType := pb.Source_SourceTypes(pb.Source_SourceTypes_value["lbry_sd_hash"])
	claim.GetStream().GetSource().SourceType = &sourceType
	src, err := hex.DecodeString(vClaim.Sources.LbrySDHash)
	claim.GetStream().GetSource().Source = src

	return claim, err
}

func migrateV2Claim(vClaim schema_version_02.Claim) (*pb.Claim, error) {

	claim := newClaim()
	//Stream
	// -->Fee
	setFee(vClaim.Fee, claim)
	// -->MetaData
	language := pb.Metadata_Language(pb.Metadata_Language_value[vClaim.Language])
	setMetaData(*claim, vClaim.Author, vClaim.Description, language,
		vClaim.License, vClaim.LicenseURL, vClaim.Title, vClaim.Thumbnail, vClaim.NSFW)
	// -->Source
	claim.GetStream().GetSource().ContentType = &vClaim.ContentType
	sourceType := pb.Source_SourceTypes(pb.Source_SourceTypes_value["lbry_sd_hash"])
	claim.GetStream().GetSource().SourceType = &sourceType
	src, err := hex.DecodeString(vClaim.Sources.LbrySDHash)
	claim.GetStream().GetSource().Source = src

	return claim, err
}

func migrateV3Claim(vClaim schema_version_03.Claim) (*pb.Claim, error) {

	claim := newClaim()
	//Stream
	// -->Fee
	setFee(vClaim.Fee, claim)
	// -->MetaData
	language := pb.Metadata_Language(pb.Metadata_Language_value[vClaim.Language])
	setMetaData(*claim, vClaim.Author, vClaim.Description, language,
		vClaim.License, vClaim.LicenseURL, vClaim.Title, vClaim.Thumbnail, vClaim.NSFW)
	// -->Source
	claim.GetStream().GetSource().ContentType = &vClaim.ContentType
	sourceType := pb.Source_SourceTypes(pb.Source_SourceTypes_value["lbry_sd_hash"])
	claim.GetStream().GetSource().SourceType = &sourceType
	src, err := hex.DecodeString(vClaim.Sources.LbrySDHash)
	claim.GetStream().GetSource().Source = src

	return claim, err
}

func setFee(fee *schema_version_01.Fee, claim *pb.Claim) {

	if fee != nil {
		amount := float32(0.0)
		currency := pb.Fee_LBC
		address := ""
		if fee.BTC != nil {
			amount = float32(fee.BTC.Amount)
			currency = pb.Fee_LBC
			address = fee.BTC.Address
		} else if fee.LBC != nil {
			amount = float32(fee.LBC.Amount)
			currency = pb.Fee_LBC
			address = fee.LBC.Address
		} else if fee.USD != nil {
			amount = float32(fee.USD.Amount)
			currency = pb.Fee_USD
			address = fee.USD.Address
		}
		//Fee Settings
		claim.GetStream().GetMetadata().GetFee().Amount = &amount
		claim.GetStream().GetMetadata().GetFee().Currency = &currency
		claim.GetStream().GetMetadata().GetFee().Address = []byte(address)
	}
}
