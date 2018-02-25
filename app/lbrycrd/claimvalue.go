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
					return nil, errors.Prefix("V3 Translation Error", err)
				}
			}
			claim, err = migrateV2Claim(*v2Claim)
			if err != nil {
				return nil, errors.Prefix("V2 Translation Error ", err)
			}
		}
		claim, err = migrateV1Claim(*v1Claim)
		if err != nil {
			return nil, errors.Prefix("V1 Translation Error ", err)
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

func migrateV1Claim(vClaim schema_version_01.Claim) (*pb.Claim, error) {

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
	//Stream
	// -->Fee
	setFee(vClaim.Fee, *claim)
	// -->MetaData
	claim.GetStream().GetMetadata().Author = &vClaim.Author
	claim.GetStream().GetMetadata().Description = &vClaim.Description
	language := pb.Metadata_Language(pb.Metadata_Language_value[vClaim.Language])
	claim.GetStream().GetMetadata().Language = &language
	claim.GetStream().GetMetadata().License = &vClaim.License
	claim.GetStream().GetMetadata().Title = &vClaim.Title
	claim.GetStream().GetMetadata().Thumbnail = &vClaim.Thumbnail
	// -->Source
	claim.GetStream().GetSource().ContentType = &vClaim.ContentType
	sourceType := pb.Source_SourceTypes(pb.Source_SourceTypes_value["lbry_sd_hash"])
	claim.GetStream().GetSource().SourceType = &sourceType
	src, _ := hex.DecodeString(vClaim.Sources.LbrySDHash)
	claim.GetStream().GetSource().Source = src
	// -->Publisher Signature
	claim.GetPublisherSignature().Signature, _ = hex.DecodeString(vClaim.PubKey)

	return claim, nil
}

func migrateV2Claim(vClaim schema_version_02.Claim) (*pb.Claim, error) {
	return nil, nil
}

func migrateV3Claim(vClaim schema_version_03.Claim) (*pb.Claim, error) {
	return nil, nil
}

func setFee(fee schema_version_01.Fee, claim pb.Claim) {
	if fee.BTC.Amount > 0 {
		amount := float32(fee.BTC.Amount)
		claim.GetStream().GetMetadata().GetFee().Amount = &amount
		currency := pb.Fee_BTC
		claim.GetStream().GetMetadata().GetFee().Currency = &currency
	} else if fee.LBC.Amount > 0 {
		amount := float32(fee.LBC.Amount)
		claim.GetStream().GetMetadata().GetFee().Amount = &amount
		currency := pb.Fee_LBC
		claim.GetStream().GetMetadata().GetFee().Currency = &currency
	} else if fee.USD.Amount > 0 {
		amount := float32(fee.USD.Amount)
		claim.GetStream().GetMetadata().GetFee().Amount = &amount
		currency := pb.Fee_USD
		claim.GetStream().GetMetadata().GetFee().Currency = &currency
	}
}
