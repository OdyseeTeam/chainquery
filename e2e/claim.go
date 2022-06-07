package e2e

import (
	"encoding/hex"

	"github.com/lbryio/chainquery/lbrycrd"
	"github.com/lbryio/chainquery/model"
	"github.com/lbryio/chainquery/util"

	"github.com/lbryio/lbry.go/v2/extras/errors"
	c "github.com/lbryio/lbry.go/v2/schema/stake"
	pb "github.com/lbryio/types/v2/go"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
)

const claimFee = 0.3

func testBasicClaimCreation() {

	_, err := lbrycrd.ClaimName("chainquery", "636861696e717565727920697320617765736f6d6521", claimFee)
	exitOnErr(errors.Err(err))
	//Legit Claims
	_, err = lbrycrd.ClaimName("example001", "7b22736f7572636573223a207b226c6272795f73645f68617368223a2022313938613661393231356435363539316235633934343634353363356361626238343236646634383335316365393866353731363562623336323337663535366263613433336636633038653635393866376234663835653733656136366638227d7d", claimFee)
	exitOnErr(errors.Err(err))
	_, err = lbrycrd.ClaimName("example002", "080110011afb03080112b303080410011a26446576696c204d6179204372792048442028772f436f6d6d656e746172792920506172742035229a02446576696c204d61792043727920506c61797468726f7567680a436f6e736f6c65202d205053332028446576696c204d61792043727920484420436f6c6c656374696f6e290a47616d65706c6179202d20687474703a2f2f7777772e54686542726f74686572686f6f646f6647616d696e672e636f6d0a506c617965727320312026203220436f6d6d656e74617279202d2057696c6c69616d204d6f72726973202620457567656e65204d6f727269730a54776974746572202d2068747470733a2f2f747769747465722e636f6d2f54424f476d6f72726973313131330a46616365626f6f6b202d2068747470733a2f2f7777772e66616365626f6f6b2e636f6d2f54424f476d6f72726973313131330a5355425343524942452a1a54686542726f74686572686f6f646f6647616d696e672e636f6d321c436f7079726967687465642028436f6e7461637420417574686f722938004a28687474703a2f2f6265726b2e6e696e6a612f7468756d626e61696c732f534e4e4c4162595331425152005a001a41080110011a30f6b79604c847c80821a2bb18a2a74c46180aaa8ae8a7731e321b9ca445e278cb81352feec9f56a9f85808dec8752ced92209766964656f2f6d70342a5c080110031a405cf7e6bded265492bf8a55434dacbfca358e5c2d1270ad8784ff4c0f3029e61af62056d7439f2296bba66edf69c52502d2c196c84d3f79e59f73911dd77dd14e2214b32a8f12c3a2e74eaad01a5ff9647aa1aa4038e0", claimFee)
	exitOnErr(errors.Err(err))
	_, err = lbrycrd.ClaimName("example003", "080110011adc010801129401080410011a0d57686174206973204c4252593f223057686174206973204c4252593f20416e20696e74726f64756374696f6e207769746820416c6578205461626172726f6b2a0c53616d75656c20427279616e32084c42525920496e6338004a2f68747470733a2f2f73332e616d617a6f6e6177732e636f6d2f66696c65732e6c6272792e696f2f6c6f676f2e706e6752005a001a41080110011a30d5169241150022f996fa7cd6a9a1c421937276a3275eb912790bd07ba7aec1fac5fd45431d226b8fb402691e79aeb24b2209766964656f2f6d7034", claimFee)
	exitOnErr(errors.Err(err))
	//Channels
	_, err = lbrycrd.ClaimName("channel001", "08011002225e0801100322583056301006072a8648ce3d020106052b8104000a0342000422ae63a64fd2cff5e698072c1a2af8117cc734b12458321946aec08081b78ce5498e9b4325b6d0352a7ab1dfe1b951a75f290f1321b26901886bb8a2fee59c0f", claimFee)
	exitOnErr(errors.Err(err))
	_, err = lbrycrd.ClaimName("channel002", "08011002225e0801100322583056301006072a8648ce3d020106052b8104000a0342000490b9d5049a72bdf7ce6b6e11f9108a3b92fcaf431d29e6040f36a2786c95e4a42a81a859fd80951f1f459113c4d781cb3647222ce0cf0ba2d117d362e823510e", claimFee)
	exitOnErr(errors.Err(err))
}

func newImageStreamClaim() (*c.StakeHelper, error) {
	streamClaim := new(pb.Claim_Stream)
	stream := new(pb.Stream)
	image := new(pb.Stream_Image)
	image.Image = new(pb.Image)
	stream.Type = image

	streamClaim.Stream = stream

	pbClaim := new(pb.Claim)
	pbClaim.Type = streamClaim

	helper := c.StakeHelper{Claim: pbClaim}

	return &helper, nil
}

func newVideoStreamClaim() (*c.StakeHelper, error) {
	streamClaim := new(pb.Claim_Stream)
	stream := new(pb.Stream)
	video := new(pb.Stream_Video)
	video.Video = new(pb.Video)
	stream.Type = video
	streamClaim.Stream = stream

	pbClaim := new(pb.Claim)
	pbClaim.Type = streamClaim

	helper := c.StakeHelper{Claim: pbClaim}

	return &helper, nil
}

func newStreamClaim(title, description string) (*c.StakeHelper, error) {
	streamClaim := new(pb.Claim_Stream)
	stream := new(pb.Stream)
	streamClaim.Stream = stream

	pbClaim := new(pb.Claim)
	pbClaim.Type = streamClaim

	helper := c.StakeHelper{Claim: pbClaim}
	helper.Claim.Title = title
	helper.Claim.Description = description

	return &helper, nil
}

func addClaimToTx(rawTx *wire.MsgTx, claim *c.StakeHelper, name string) error {

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

func signClaim(rawTx *wire.MsgTx, privKey btcec.PrivateKey, claim, channel *c.StakeHelper, channelClaimID string) error {
	claimIDHexBytes, err := hex.DecodeString(channelClaimID)
	if err != nil {
		return errors.Err(err)
	}
	claim.Version = c.WithSig
	claim.ClaimID = util.ReverseBytes(claimIDHexBytes)
	hash, err := c.GetOutpointHash(rawTx.TxIn[0].PreviousOutPoint.Hash.String(), rawTx.TxIn[0].PreviousOutPoint.Index)
	if err != nil {
		return err
	}
	sig, err := c.Sign(privKey, *channel, *claim, hash)
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
			return errors.Err(err)
		}
		if !claim.IsCertValid {
			return errors.Err("claimname %s does not have a valid cert when expected", name)
		}
	}
	return nil
}
