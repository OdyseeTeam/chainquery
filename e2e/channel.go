package e2e

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/lbryio/lbry.go/extras/errors"
	"github.com/lbryio/lbry.go/v2/schema/keys"
	c "github.com/lbryio/lbry.go/v2/schema/stake"
	pb "github.com/lbryio/types/v2/go"
)

func newChannel() (*c.StakeHelper, *btcec.PrivateKey, error) {
	claimChannel := new(pb.Claim_Channel)
	channel := new(pb.Channel)
	claimChannel.Channel = channel

	pbClaim := new(pb.Claim)
	pbClaim.Type = claimChannel

	privateKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, nil, errors.Err(err)
	}
	pubkeyBytes, err := keys.PublicKeyToDER(privateKey.PubKey())
	if err != nil {
		return nil, nil, errors.Err(err)
	}

	helper := c.StakeHelper{Claim: pbClaim}
	helper.Version = c.NoSig
	helper.Claim.GetChannel().PublicKey = pubkeyBytes
	helper.Claim.Tags = []string{"Foo", "Bar"}
	coverSrc := new(pb.Source)
	coverSrc.Url = "https://coverurl.com"
	helper.Claim.GetChannel().Cover = coverSrc
	helper.Claim.GetChannel().WebsiteUrl = "https://homepageurl.com"
	helper.Claim.Languages = []*pb.Language{{Language: pb.Language_en}}
	helper.Claim.Title = "title"
	helper.Claim.Description = "description"
	thumbnailSrc := new(pb.Source)
	thumbnailSrc.Url = "thumbnailurl.com"
	helper.Claim.Thumbnail = thumbnailSrc
	helper.Claim.Locations = []*pb.Location{{Country: pb.Location_US}}

	return &helper, privateKey, nil
}

func createChannel(name string) (*c.StakeHelper, *btcec.PrivateKey, error) {
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
