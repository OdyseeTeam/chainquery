package jobs

import (
	"context"
	"time"

	"github.com/lbryio/chainquery/global"
	"github.com/lbryio/chainquery/metrics"
	"github.com/lbryio/chainquery/model"

	"github.com/lbryio/lbry.go/extras/errors"
	c "github.com/lbryio/lbryschema.go/claim"

	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/volatiletech/sqlboiler/queries"
)

const certificateSyncPrefix = "Certificate Sync:"

var certificateSyncRunning = false

const certsProcessedPerIteration = 1000

//CertificateSync processed all claims that have not been processed yet and verifies that any claims for channels, are
// signed by the channels certificate. This ensure that the channel owner actually published this claim.
func CertificateSync() {
	if !certificateSyncRunning {
		metrics.JobLoad.WithLabelValues("certificate_sync").Inc()
		defer metrics.JobLoad.WithLabelValues("certificate_sync").Dec()
		defer metrics.Job(time.Now(), "certificate_sync")
		logrus.Debug("Running Certificate Sync...")
		certificateSyncRunning = true
		defer endCertificateSync()
		claims, err := getClaimsToBeSynced()
		if err != nil {
			logrus.Error(certificateSyncPrefix+" Unable to get claims that need certificates checked", errors.Err(err))
		}
		for _, claimToBeSynced := range claims {
			err := claimToBeSynced.populateFirstInputInfo()
			if err != nil {
				logrus.Error(certificateSyncPrefix+"", err)
				continue
			}
			claim := model.Claim{ID: claimToBeSynced.ID}
			certified, err := certifyClaim(claimToBeSynced)
			if err != nil {
				logrus.Error(certificateSyncPrefix+" [claim.id= ", claimToBeSynced.ID, "]", errors.Err(err))
			}
			claim.IsCertProcessed = true
			if certified {
				claim.IsCertValid = true
				err := claim.UpdateG(boil.Whitelist(model.ClaimColumns.IsCertValid, model.ClaimColumns.IsCertProcessed))
				if err != nil {
					logrus.Error(certificateSyncPrefix+" [claim.id= ", claimToBeSynced.ID, "]", errors.Err(err))
				}
				continue
			}
			err = claim.UpdateG(boil.Whitelist(model.ClaimColumns.IsCertProcessed))
			if err != nil {
				logrus.Error(certificateSyncPrefix, errors.Err(err))
			}
		}
	}
}

func endCertificateSync() {
	certificateSyncRunning = false
}

func certifyClaim(claimToBeSynced claimToBeSynced) (bool, error) {

	signedHelper, err := c.DecodeClaimHex(claimToBeSynced.SignedClaimHex, global.BlockChainName)
	if err != nil {
		return false, errors.Err(errors.Prefix(certificateSyncPrefix, err))
	}

	certHelper, err := c.DecodeClaimHex(claimToBeSynced.ChannelHex, global.BlockChainName)
	if err != nil {
		return false, errors.Err(errors.Prefix(certificateSyncPrefix, err))
	}

	if claimToBeSynced.FirstInputTxHash != "" {
		firstInputHash, err := c.GetOutpointHash(claimToBeSynced.FirstInputTxHash, uint32(claimToBeSynced.FirstInputTxOPosition))
		if err != nil {
			return false, err
		}
		if verified, err := signedHelper.ValidateClaimSignature(certHelper, firstInputHash, claimToBeSynced.ChannelClaimID, global.BlockChainName); verified {
			return verified, err
		}
	}

	return signedHelper.ValidateClaimSignature(certHelper, claimToBeSynced.SignedClaimAddress, claimToBeSynced.ChannelClaimID, global.BlockChainName)
}

type claimToBeSynced struct {
	ID                    uint64 `boil:"id"`
	SignedClaimHex        string `boil:"signed_claim_hex"`
	SignedClaimAddress    string `boil:"claim_address"`
	TransactionHashUpdate string `boil:"transaction_hash_update"`
	ChannelHex            string `boil:"channel_hex"`
	ChannelClaimID        string `boil:"claim_id"`
	FirstInputTxHash      string `boil:"first_input_tx_hash"`
	FirstInputTxOPosition uint64 `boil:"first_input_txo_position"`
}

func (c *claimToBeSynced) populateFirstInputInfo() error {
	w := model.InputWhere
	input, err := model.Inputs(w.TransactionHash.EQ(c.TransactionHashUpdate), w.Vin.EQ(null.UintFrom(0))).OneG()
	if err != nil {
		return errors.Err(err)
	}
	if input != nil {
		c.FirstInputTxHash = input.PrevoutHash.String
		c.FirstInputTxOPosition = uint64(input.PrevoutN.Uint)
	}
	return nil
}

func getClaimsToBeSynced() ([]claimToBeSynced, error) {
	var context context.Context
	claim := model.TableNames.Claim
	claimID := claim + "." + model.ClaimColumns.ID
	signedClaimHex := claim + "." + model.ClaimColumns.ValueAsHex + " as signed_claim_hex"
	claimAddress := claim + "." + model.ClaimColumns.ClaimAddress
	claimTxHashUpdate := claim + "." + model.ClaimColumns.TransactionHashUpdate
	channelHex := "channel." + model.ClaimColumns.ValueAsHex + " as channel_hex"
	ChannelClaimID := "channel." + model.ClaimColumns.ClaimID
	publisherID := claim + "." + model.ClaimColumns.PublisherID
	isCertProcessed := claim + "." + model.ClaimColumns.IsCertProcessed

	var claims []claimToBeSynced
	err := queries.Raw(`
		SELECT 
			`+claimID+`,
			`+signedClaimHex+`,
			`+claimAddress+`,
			`+channelHex+`,
			`+ChannelClaimID+`,
			`+claimTxHashUpdate+`
		FROM `+claim+`
		INNER JOIN `+claim+` channel ON `+ChannelClaimID+` = `+publisherID+` 
		WHERE `+isCertProcessed+`=? LIMIT ?`, false, certsProcessedPerIteration).BindG(context, &claims)
	if err != nil {
		return nil, err
	}

	return claims, nil

}
