package processing

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/lbryio/chainquery/datastore"
	"github.com/lbryio/chainquery/global"
	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/metrics"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/notifications"
	"github.com/lbryio/chainquery/sockety"
	util2 "github.com/lbryio/chainquery/util"

	"github.com/lbryio/lbry.go/extras/errors"
	util "github.com/lbryio/lbry.go/lbrycrd"
	"github.com/lbryio/lbryschema.go/address/base58"
	c "github.com/lbryio/lbryschema.go/claim"
	"github.com/lbryio/sockety/socketyapi"
	pb "github.com/lbryio/types/v2/go"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
)

func processAsClaim(script []byte, vout model.Output, tx model.Transaction, blockHeight uint64) (address *string, claimID *string, err error) {
	defer metrics.Processing(time.Now(), "claim")
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
	if claim == nil {
		claim = &model.Claim{ClaimID: claimid, TransactionHashID: null.NewString(tx.Hash, true), Vout: vout.Vout}
		err := datastore.PutClaim(claim)
		if err != nil {
			return name, claimid, pkscript, err
		}
	}
	claim, err = processClaim(helper, claim, value, vout, tx)
	if err != nil {
		return name, claimid, pkscript, err
	}
	claim.ClaimID = claimid
	claim.Name = name
	claim.TransactionTime = tx.TransactionTime
	claim.ClaimAddress = lbrycrd.GetAddressFromPublicKeyScript(pkscript)
	claim.TransactionHashUpdate.SetValid(tx.Hash)
	claim.VoutUpdate.SetValid(vout.Vout)
	if blockHeight > 0 {
		claim.Height = uint(blockHeight)
	} else {
		logrus.Debug("ClaimNew: No blockheight!")
	}
	err = datastore.PutClaim(claim)
	if err == nil {
		sockety.SendNotification(socketyapi.SendNotificationArgs{
			Service: socketyapi.BlockChain,
			Type:    "new_claim",
			IDs:     []string{"claims", claim.Name, claimid},
			Data:    map[string]interface{}{"claim": claim},
		})
	}

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
	} else {
		sockety.SendNotification(socketyapi.SendNotificationArgs{
			Service: socketyapi.BlockChain,
			Type:    "support",
			IDs:     []string{"supports", claimid, name},
			Data:    map[string]interface{}{"support": support},
		})
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
		claim, err := processUpdateClaim(helper, claim, value)
		if err != nil {
			return name, claimID, pubkeyscript, err
		}
		if claim == nil {
			claim = &model.Claim{Name: name, ClaimID: claimID, TransactionHashID: null.NewString(tx.Hash, true), Vout: vout.Vout}
			err := datastore.PutClaim(claim)
			if err != nil {
				return name, claimID, pubkeyscript, err
			}
		}
		claim.TransactionTime = tx.TransactionTime
		claim.ClaimAddress = lbrycrd.GetAddressFromPublicKeyScript(pubkeyscript)
		if blockHeight > 0 {
			claim.Height = uint(blockHeight)
		} else {
			logrus.Debug("ClaimUpdate: No blockheight!")
		}
		claim.TransactionHashUpdate.SetValid(tx.Hash)
		claim.VoutUpdate.SetValid(vout.Vout)
		if claim.BidState == "Spent" {
			claim.BidState = "Accepted"
		}
		if err := datastore.PutClaim(claim); err != nil {
			logrus.Debug("Claim updates to invalid certificate claim. ", claim.PublisherID)
			if logrus.GetLevel() == logrus.DebugLevel {
				logrus.WithError(err)
			}
		} else {
			sockety.SendNotification(socketyapi.SendNotificationArgs{
				Service: socketyapi.BlockChain,
				Type:    "claim_update",
				IDs:     []string{"claims", "claimupdates", claim.ClaimID, name},
				Data:    map[string]interface{}{"claim": claim},
			})
		}
	}
	return name, claimID, pubkeyscript, err
}

func processClaim(helper *c.ClaimHelper, claim *model.Claim, value []byte, output model.Output, tx model.Transaction) (*model.Claim, error) {
	claim.ValueAsHex = hex.EncodeToString(value)
	if helper.GetStream() != nil {
		claim.ClaimType = 1
	} else if helper.GetChannel() != nil {
		claim.ClaimType = 2
	}

	// pbClaim JSON
	if claimHelper, err := c.DecodeClaimHex(claim.ValueAsHex, global.BlockChainName); err == nil && claimHelper != nil {
		json, err := GetValueAsJSON(*claimHelper)
		if err != nil {
			logrus.Error(err)
		} else {
			claim.ValueAsJSON.SetValid(json)
		}
	}

	setSourceInfo(claim, helper)
	err := setMetaDataInfo(claim, helper)
	if err != nil {
		return nil, err
	}
	setPublisherInfo(claim, helper)
	setCertificateInfo(claim, helper)

	if helper.LegacyClaim != nil {
		claim.Version = helper.LegacyClaim.GetVersion().String()
	}
	notifications.ClaimEvent(claim.ClaimID, claim.Name, claim.Title.String, tx.Hash, claim.PublisherID.String, claim.SourceHash.String)
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

func processUpdateClaim(helper *c.ClaimHelper, claim *model.Claim, value []byte) (*model.Claim, error) {
	if claim == nil {
		return nil, nil
	}
	claim.ValueAsHex = hex.EncodeToString(value)

	err := UpdateClaimData(helper, claim)
	if err != nil {
		return nil, err
	}

	return claim, nil
}

// UpdateClaimData updates the claim information from the blockchain
func UpdateClaimData(helper *c.ClaimHelper, claim *model.Claim) error {
	// pbClaim JSON
	if claimHelper, err := c.DecodeClaimHex(claim.ValueAsHex, global.BlockChainName); err == nil && claimHelper != nil {
		json, err := GetValueAsJSON(*claimHelper)
		if err != nil {
			logrus.Error(err)
		} else {
			claim.ValueAsJSON.SetValid(json)
		}
	}

	setSourceInfo(claim, helper)
	err := setMetaDataInfo(claim, helper)
	if err != nil {
		return err
	}
	setPublisherInfo(claim, helper)
	setCertificateInfo(claim, helper)

	if helper.LegacyClaim != nil {
		claim.Version = helper.LegacyClaim.GetVersion().String()
	}
	return nil
}

func setPublisherInfo(claim *model.Claim, helper *c.ClaimHelper) {
	claim.IsCertProcessed = true
	claim.IsCertValid = false
	claim.PublisherID = null.NewString("", false)
	claim.PublisherSig = null.NewString("", false)
	if helper.Signature != nil {
		claim.IsCertProcessed = false
		if helper.LegacyClaim == nil {
			claim.PublisherID.SetValid(hex.EncodeToString(util2.ReverseBytes(helper.ClaimID)))
		} else {
			claim.PublisherID.SetValid(hex.EncodeToString(helper.ClaimID))
		}
		claim.PublisherSig.SetValid(hex.EncodeToString(helper.Signature))
	}
}

func setCertificateInfo(claim *model.Claim, helper *c.ClaimHelper) {
	claim.Certificate = null.NewString("", false)
	if helper.GetChannel() != nil {
		claim.IsCertProcessed = true
		if helper.LegacyClaim != nil {
			certificate := helper.LegacyClaim.GetCertificate()
			certBytes, err := json.Marshal(certificate)
			if err != nil {
				logrus.Error("Could not form json from certificate")
			}
			claim.Certificate.SetValid(string(certBytes))
		}
	}
}

func setMetaDataInfo(claim *model.Claim, helper *c.ClaimHelper) error {
	err := resetMetadata(claim)
	if err != nil {
		return err
	}
	claim.Title.SetValid(helper.GetTitle())
	claim.Description.SetValid(helper.GetDescription())
	claim.ThumbnailURL.SetValid(helper.GetThumbnail().GetUrl())
	if len(helper.GetTags()) > 0 {
		err := setTags(claim, helper.GetTags())
		if err != nil {
			return err
		}
	}
	if len(helper.GetLanguages()) > 0 {
		claim.Language.SetValid(helper.GetLanguages()[0].Language.String())
	}
	stream := helper.GetStream()
	if stream != nil {
		setStreamMetadata(claim, *stream)
	}
	channel := helper.GetChannel()
	if channel != nil {
		setChannelMetadata(claim, *channel)
	}
	list := helper.GetCollection()
	if list != nil {
		setCollectionMetadata(claim, *list)
	}
	reference := helper.GetRepost()
	if reference != nil {
		claim.Type.SetValid(global.ClaimReferenceClaimType)
		if len(reference.GetClaimHash()) > 0 {
			claim.ClaimReference.SetValid(hex.EncodeToString(util2.ReverseBytes(reference.GetClaimHash())))
		}
	}

	return nil
}

func setTags(claim *model.Claim, tags []string) error {
	maxTagLength := 255
	for _, tag := range tags {
		if len(tag) > maxTagLength {
			tag = tag[0:maxTagLength]
		}
		if tag == "mature" {
			claim.IsNSFW = true
		}
		t := datastore.GetTag(tag)
		if t == nil {
			t = &model.Tag{Tag: tag}
			err := datastore.PutTag(t)
			if err != nil {
				logrus.Error(errors.Prefix(fmt.Sprintf("Could not save tag %s, skipping: ", tag), err))
				return nil
			}
		}
		ct := datastore.GetClaimTag(t.ID, claim.ClaimID)
		if ct == nil {
			ct = &model.ClaimTag{ClaimID: claim.ClaimID, TagID: null.NewUint64(t.ID, true)}
			err := datastore.PutClaimTag(ct)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func setLicense(claim *model.Claim, stream pb.Stream) {
	license := stream.GetLicense()
	if len([]rune(license)) > 500 {
		license = string([]rune(license)[:500])
	}
	if utf8.ValidString(license) {
		claim.License.SetValid(license)
	}

	liscenseURL := stream.GetLicenseUrl()
	if len(liscenseURL) > 255 {
		liscenseURL = liscenseURL[0:255]
	}
	claim.LicenseURL.SetValid(liscenseURL)
}

func setStreamMetadata(claim *model.Claim, stream pb.Stream) {
	claim.Type.SetValid(global.StreamClaimType)
	claim.Author.SetValid(stream.GetAuthor())
	setLicense(claim, stream)
	claim.Preview.SetValid("") //Never set

	fee := stream.GetFee()
	if fee != nil {
		claim.FeeCurrency.SetValid(fee.GetCurrency().String())
		claim.Fee = float64(fee.GetAmount())
		claim.FeeAddress = base58.EncodeBase58(fee.GetAddress())
	}
	s := stream.GetSource()
	if s != nil {
		setSourceMetadata(claim, s)
	}
	if stream.GetReleaseTime() > 0 {
		claim.ReleaseTime.SetValid(uint64(stream.GetReleaseTime()))
	}
	if stream.GetImage() != nil {
		i := stream.GetImage()
		if i.GetHeight() > 0 {
			claim.FrameHeight.SetValid(uint64(i.GetHeight()))
		}
		if i.GetWidth() > 0 {
			claim.FrameWidth.SetValid(uint64(i.GetWidth()))
		}
	}
	if stream.GetVideo() != nil {
		v := stream.GetVideo()
		if v.GetHeight() > 0 {
			claim.FrameHeight.SetValid(uint64(v.GetHeight()))
		}
		if v.GetWidth() > 0 {
			claim.FrameWidth.SetValid(uint64(v.GetWidth()))
		}
		if v.GetDuration() > 0 {
			claim.Duration.SetValid(uint64(v.GetDuration()))
		}
		if v.GetAudio() != nil {
			if v.GetAudio().GetDuration() > 0 {
				claim.AudioDuration.SetValid(uint64(v.GetAudio().GetDuration()))
			}
		}
	}
	if stream.GetAudio() != nil {
		if stream.GetAudio().GetDuration() > 0 {
			claim.AudioDuration.SetValid(uint64(stream.GetAudio().GetDuration()))
		}
	}
	if stream.GetSoftware() != nil {
		s := stream.GetSoftware()
		if s.GetOs() != "" {
			claim.Os.SetValid(s.GetOs())
		}
	}
}

func setChannelMetadata(claim *model.Claim, channel pb.Channel) {
	claim.Type.SetValid(global.ChannelClaimType)
	if channel.GetCover() != nil {
		c := channel.GetCover()
		if c.GetName() != "" {
			claim.SourceName.SetValid(c.GetName())
		}
		if c.GetSize() > 0 {
			claim.SourceSize.SetValid(c.GetSize())
		}
		if c.GetUrl() != "" {
			const maxSourceURLLength = 255
			sourceURL := c.GetUrl()
			if len(sourceURL) > maxSourceURLLength {
				sourceURL = sourceURL[:252] + "..."
			}
			claim.SourceURL.SetValid(sourceURL)
		}
		if len(c.GetHash()) > 0 {
			claim.SourceHash.SetValid(hex.EncodeToString(c.GetHash()))
		}
		if c.GetMediaType() != "" {
			claim.SourceMediaType.SetValid(c.GetMediaType())
		}
	}
	if channel.GetEmail() != "" {
		claim.Email.SetValid(channel.GetEmail())
	}
	if channel.GetFeatured() != nil {
		claim.HasClaimList.SetValid(true)
		claim.ListType.SetValid(int16(channel.GetFeatured().GetListType()))
		claimList := make([]string, len(channel.GetFeatured().GetClaimReferences()))
		for i, c := range channel.GetFeatured().GetClaimReferences() {
			// No need to reverse bytes as lbrynet is fixed and should do this now
			claimList[i] = hex.EncodeToString(c.ClaimHash)
		}
		jsonList, err := json.Marshal(claimList)
		if err == nil {
			claim.ClaimIDList.SetValid(jsonList)
		} else {
			logrus.Error("could not process claim list of channel [", claim.ClaimID, "]")
		}
		//ToDo - Create NM Table Entry for each
	}
}

func setCollectionMetadata(claim *model.Claim, list pb.ClaimList) {
	claim.Type.SetValid(global.ClaimListClaimType)
	claim.HasClaimList.SetValid(true)
	claim.ListType.SetValid(int16(list.GetListType()))
	claimList := make([]string, len(list.GetClaimReferences()))
	for i, c := range list.GetClaimReferences() {
		// No need to reverse bytes as lbrynet is fixed and should do this now
		claimList[i] = hex.EncodeToString(c.ClaimHash)
	}
	jsonList, err := json.Marshal(claimList)
	if err == nil {
		claim.ClaimIDList.SetValid(jsonList)
	} else {
		logrus.Error("could not process claim list of channel [", claim.ClaimID, "]")
	}
	//ToDo - Create NM Table Entry for each
}

func setSourceMetadata(claim *model.Claim, s *pb.Source) {
	if s.GetUrl() != "" {
		const maxSourceURLLength = 255
		sourceURL := s.GetUrl()
		if len(sourceURL) > maxSourceURLLength {
			sourceURL = sourceURL[:252] + "..."
		}
		claim.SourceURL.SetValid(sourceURL)
	}
	if len(s.GetHash()) > 0 {
		claim.SourceHash.SetValid(hex.EncodeToString(s.GetHash()))
	}
	if s.GetSize() > 0 {
		claim.SourceSize.SetValid(s.GetSize())
	}
	if s.GetName() != "" {
		claim.SourceName.SetValid(s.GetName())
	}
	if s.GetMediaType() != "" {
		claim.SourceMediaType.SetValid(s.GetMediaType())
	}
}

func resetMetadata(claim *model.Claim) error {
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
	claim.Type = null.NewString("", false)
	claim.ReleaseTime = null.NewUint64(0, false)
	claim.SourceHash = null.NewString("", false)
	claim.SourceName = null.NewString("", false)
	claim.SourceSize = null.NewUint64(0, false)
	claim.SourceMediaType = null.NewString("", false)
	claim.SourceURL = null.NewString("", false)
	claim.FrameWidth = null.NewUint64(0, false)
	claim.FrameHeight = null.NewUint64(0, false)
	claim.Duration = null.NewUint64(0, false)
	claim.AudioDuration = null.NewUint64(0, false)
	claim.Os = null.NewString("", false)
	claim.Email = null.NewString("", false)
	claim.WebsiteURL = null.NewString("", false)
	claim.HasClaimList = null.NewBool(false, false)
	claim.ClaimReference = null.NewString("", false)
	claim.ListType = null.NewInt16(0, false)
	claim.ClaimIDList = null.NewJSON(nil, false)
	claim.Country = null.NewString("", false)
	claim.State = null.NewString("", false)
	claim.Code = null.NewString("", false)
	claim.City = null.NewString("", false)
	claim.Longitude = null.NewInt64(0, false)
	claim.Latitude = null.NewInt64(0, false)

	err := claim.ListClaimClaimInLists().DeleteAll(boil.GetDB())
	if err != nil {
		return err
	}
	err = claim.ClaimTags().DeleteAll(boil.GetDB())
	if err != nil {
		return err
	}

	return nil
}

func setSourceInfo(claim *model.Claim, helper *c.ClaimHelper) {
	claim.ContentType = null.NewString("", false)
	claim.SDHash = null.NewString("", false)
	stream := helper.GetStream()
	if stream != nil {
		source := stream.GetSource()
		if source != nil {
			const maxContentTypeLength = 162
			mediaType := source.GetMediaType()
			if len(mediaType) > maxContentTypeLength {
				mediaType = mediaType[:158] + "..."
			}
			claim.ContentType.SetValid(mediaType)
			sdHash := hex.EncodeToString(stream.GetSource().GetSdHash())
			const maxSDHashColLength = 120
			if len(sdHash) > maxSDHashColLength {
				sdHash = sdHash[:116] + "..."
			}
			claim.SDHash.SetValid(sdHash)
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
