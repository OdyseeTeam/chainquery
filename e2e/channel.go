package e2e

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/lbryio/lbry.go/extras/errors"
	c "github.com/lbryio/lbryschema.go/claim"
	pb "github.com/lbryio/types/v2/go"
)

func newChannel() (*c.ClaimHelper, *btcec.PrivateKey, error) {
	claimChannel := new(pb.Claim_Channel)
	channel := new(pb.Channel)
	claimChannel.Channel = channel

	pbClaim := new(pb.Claim)
	pbClaim.Type = claimChannel

	privateKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, errors.Err(err)
	}
	pubkeyBytes, err := c.PublicKeyToDER(privateKey.PubKey())
	if err != nil {
		return nil, nil, errors.Err(err)
	}

	helper := c.ClaimHelper{Claim: pbClaim}
	helper.Version = c.NoSig
	helper.GetChannel().PublicKey = pubkeyBytes
	helper.Tags = []string{"Foo", "Bar"}
	coverSrc := new(pb.Source)
	coverSrc.Url = "https://coverurl.com"
	helper.GetChannel().Cover = coverSrc
	helper.GetChannel().WebsiteUrl = "https://homepageurl.com"
	helper.Languages = []*pb.Language{{Language: pb.Language_en}}
	helper.Title = "title"
	helper.Description = "description"
	thumbnailSrc := new(pb.Source)
	thumbnailSrc.Url = "thumbnailurl.com"
	helper.Thumbnail = thumbnailSrc
	helper.Locations = []*pb.Location{{Country: pb.Location_US}}

	return &helper, privateKey, nil
}

func createChannel(name string) (*c.ClaimHelper, *btcec.PrivateKey, error) {
	channel, key, err := newChannel()
	if err != nil {
		return nil, nil, err
	}

	rawTx, err := getEmptyTx(claimAmount)
	if err != nil {
		return nil, nil, err
	}
	err = addClaimToTx(rawTx, channel, name)
	if err != nil {
		return nil, nil, err
	}

	_, err = signTxAndSend(rawTx)
	if err != nil {
		return nil, nil, err
	}

	return channel, key, nil
}
