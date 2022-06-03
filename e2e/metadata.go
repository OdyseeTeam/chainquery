package e2e

import (
	"crypto/sha256"
	"time"

	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/lbry.go/v2/extras/errors"
	"github.com/lbryio/lbry.go/v2/schema/stake"
	pb "github.com/lbryio/types/v2/go"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

func testVideoMetaData() {
	pbClaim, err := newVideoStreamClaim()
	exitOnErr(errors.Err(err))
	pbClaim.Claim.Title = "Angry Tom Fights Back!"
	pbClaim.Claim.Description = "Long ago in a galaxy far far away, Tom came up with an idea to transform the game of angry birds."
	pbClaim.Claim.Tags = []string{"mature", "tom", "angry", "angry birds"}
	pbClaim.Claim.Thumbnail = new(pb.Source)
	pbClaim.Claim.GetThumbnail().Url = "http://www.lbry.com/images/angrytompreview.jpeg"
	pbClaim.GetStream().Author = "Mark Beamer Jr"
	pbClaim.GetStream().GetVideo().Height = 100
	pbClaim.GetStream().GetVideo().Width = 200
	pbClaim.GetStream().GetVideo().Audio = new(pb.Audio)
	pbClaim.GetStream().GetVideo().GetAudio().Duration = 1200
	pbClaim.GetStream().ReleaseTime = time.Now().Unix()
	pbClaim.GetStream().Source = new(pb.Source)
	pbClaim.GetStream().GetSource().Size = 1500
	sha := sha256.Sum256([]byte("sdhash"))
	pbClaim.GetStream().GetSource().SdHash = sha[:]
	pbClaim.GetStream().GetSource().Url = "MySourceURL"
	pbClaim.GetStream().GetSource().MediaType = "video"
	pbClaim.GetStream().GetSource().Name = "angrytom.mpeg"
	sha = sha256.Sum256([]byte("hash"))
	pbClaim.GetStream().GetSource().Hash = sha[:]
	pbClaim.Version = stake.NoSig

	test(func() {
		rawTx, err := getEmptyTx(claimAmount * 1)
		exitOnErr(errors.Err(err))
		exitOnErr(addClaimToTx(rawTx, pbClaim, "testvideo"))
		_, err = signTxAndSend(rawTx)
		exitOnErr(errors.Err(err))
	}, func() error {
		claim, err := model.Claims(qm.Where("name = ?", "testvideo")).OneG()
		exitOnErr(errors.Err(err))
		verify(claim.Title.String == pbClaim.Claim.GetTitle(), "Title does not match after publishing")
		verify(claim.Description.String == pbClaim.Claim.GetDescription(), "Description doesnt match")
		verify(claim.Name == "testvideo", "Claim name doesnt match")
		verify(claim.AudioDuration.Uint64 == uint64(pbClaim.GetStream().GetVideo().GetAudio().Duration), "Video audio duration doesnt match")
		verify(claim.FrameHeight.Uint64 == uint64(pbClaim.GetStream().GetVideo().GetHeight()), "Video height doesnt match")
		verify(claim.FrameWidth.Uint64 == uint64(pbClaim.GetStream().GetVideo().GetWidth()), "Video width doesnt match")
		return nil
	}, 1)
}

func testImageMetadata() {
	pbClaim, err := newImageStreamClaim()
	exitOnErr(errors.Err(err))
	pbClaim.Claim.Title = "Angry Tom Fights Back!"
	pbClaim.Claim.Description = "Long ago in a galaxy far far away, Tom came up with an idea to transform the game of angry birds."
	pbClaim.Claim.Tags = []string{"mature", "tom", "angry", "angry birds"}
	pbClaim.Claim.Thumbnail = new(pb.Source)
	pbClaim.Claim.GetThumbnail().Url = "http://www.lbry.com/images/angrytompreview.jpeg"
	pbClaim.GetStream().Author = "Mark Beamer Jr"
	pbClaim.GetStream().GetImage().Height = 100
	pbClaim.GetStream().GetImage().Width = 200
	pbClaim.GetStream().ReleaseTime = time.Now().Unix()
	pbClaim.GetStream().Source = new(pb.Source)
	pbClaim.GetStream().GetSource().Size = 1500
	sha := sha256.Sum256([]byte("sdhash"))
	pbClaim.GetStream().GetSource().SdHash = sha[:]
	pbClaim.GetStream().GetSource().Url = "MySourceURL"
	pbClaim.GetStream().GetSource().MediaType = "video"
	pbClaim.GetStream().GetSource().Name = "angrytom.mpeg"
	sha = sha256.Sum256([]byte("hash"))
	pbClaim.GetStream().GetSource().Hash = sha[:]
	pbClaim.Version = stake.NoSig

	test(func() {
		rawTx, err := getEmptyTx(claimAmount * 1)
		exitOnErr(errors.Err(err))
		exitOnErr(addClaimToTx(rawTx, pbClaim, "testimage"))
		_, err = signTxAndSend(rawTx)
		exitOnErr(errors.Err(err))
	}, func() error {
		claim, err := model.Claims(qm.Where("name = ?", "testimage")).OneG()
		exitOnErr(errors.Err(err))
		verify(claim.Title.String == pbClaim.Claim.GetTitle(), "Title does not match after publishing")
		verify(claim.Description.String == pbClaim.Claim.GetDescription(), "Description doesnt match")
		verify(claim.Name == "testimage", "Claim name doesnt match")
		return nil
	}, 1)
}
