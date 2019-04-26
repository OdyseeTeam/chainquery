package e2e

import (
	"encoding/hex"

	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"

	"github.com/lbryio/lbry.go/extras/errors"
	c "github.com/lbryio/lbryschema.go/claim"
	pb "github.com/lbryio/types/v2/go"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/volatiletech/sqlboiler/queries/qm"
)

func newClaim(title, description string) (*c.ClaimHelper, error) {
	streamClaim := new(pb.Claim_Stream)
	stream := new(pb.Stream)
	streamClaim.Stream = stream

	pbClaim := new(pb.Claim)
	pbClaim.Type = streamClaim

	helper := c.ClaimHelper{Claim: pbClaim}
	helper.Title = title
	helper.Description = description

	return &helper, nil

}

func addClaimToTx(rawTx *wire.MsgTx, claim *c.ClaimHelper, name string) error {

	address, err := lbrycrd.LBRYcrdClient.GetNewAddress("")
	if err != nil {
		return errors.Err(err)
	}
	amount, err := btcutil.NewAmount(claimAmount)
	if err != nil {
		return errors.Err(err)
	}

	value, err := claim.CompileValue()
	if err != nil {
		return errors.Err(err)
	}
	script, err := getClaimNamePayoutScript(name, value, address)
	if err != nil {
		return errors.Err(err)
	}
	rawTx.AddTxOut(wire.NewTxOut(int64(amount), script))

	return nil
}

func signClaim(rawTx *wire.MsgTx, privKey btcec.PrivateKey, claim, channel *c.ClaimHelper, channelClaimID string) error {
	claimIDHexBytes, err := hex.DecodeString(channelClaimID)
	if err != nil {
		return err
	}
	claim.Version = c.WithSig
	claim.ClaimID = util.ReverseBytes(claimIDHexBytes)
	sig, err := c.Sign(privKey, *channel, *claim, rawTx.TxIn[0].PreviousOutPoint.Hash.String())
	if err != nil {
		return err
	}

	lbrySig, err := sig.LBRYSDKEncode()
	if err != nil {
		return err
	}
	claim.Signature = lbrySig

	return nil

}

func checkCertValid(claimNames []string) error {
	for _, name := range claimNames {
		claim, err := model.Claims(qm.Where("name=?", name)).OneG()
		if err != nil {
			return err
		}
		if !claim.IsCertValid {
			return errors.Err("claimname %s does not have a valid cert when expected", name)
		}
	}
	return nil
}
