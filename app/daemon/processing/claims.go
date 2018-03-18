package processing

import (
	"encoding/hex"
	"fmt"
	"github.com/lbryio/chainquery/app/lbrycrd"
	"github.com/lbryio/chainquery/app/model"
	"github.com/lbryio/errors.go"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ripemd160"
	"strconv"
	"strings"
)

func processAsClaim(script []byte, vout model.Output) (address *string, err error) {
	var pubkeyscript []byte
	var name string
	var claimid string
	if lbrycrd.IsClaimNameScript(script) {
		name, claimid, pubkeyscript, err = processClaimNameScript(&script, vout)
		if err != nil {
			return nil, err
		}
		return nil, nil
	} else if lbrycrd.IsClaimSupportScript(script) {
		name, claimid, pubkeyscript, err = processClaimSupportScript(&script, vout)
		if err != nil {
			return nil, err
		}
		return nil, nil
	} else if lbrycrd.IsClaimUpdateScript(script) {
		name, claimid, pubkeyscript, err = processClaimUpdateScript(&script, vout)
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

func processClaimNameScript(script *[]byte, vout model.Output) (name string, claimid string, pkscript []byte, err error) {
	name, value, pubkeyscript, err := lbrycrd.ParseClaimNameScript(*script)
	if err != nil {
		errors.Prefix("Claim name processing error: ", err)
		return name, claimid, pubkeyscript, err
	}
	_, err = lbrycrd.DecodeClaimValue(name, value)
	if false { //claim != nil {
		hasher := ripemd160.New()
		value := strconv.Itoa(int(vout.TransactionID)) + strconv.Itoa(int(vout.Vout))
		hasher.Write([]byte(value))
		hashBytes := hasher.Sum(nil)
		claimId := fmt.Sprintf("%x", hashBytes)
		if claimId != "" {
			//log.Debug("ClaimName ", name, " ClaimId ", claimId)
		}
	}

	return name, claimid, pubkeyscript, err
}

func processClaimSupportScript(script *[]byte, vout model.Output) (name string, claimid string, pubkeyscript []byte, err error) {
	name, claimid, pubkeyscript, err = lbrycrd.ParseClaimSupportScript(*script)
	if err != nil {
		errors.Prefix("Claim support processing error: ", err)
		return name, claimid, pubkeyscript, err
	}
	//log.Debug("ClaimSupport ", name, " ClaimId ", claimid)

	return name, claimid, pubkeyscript, err
}

func processClaimUpdateScript(script *[]byte, vout model.Output) (name string, claimId string, pubkeyscript []byte, err error) {
	name, claimId, value, pubkeyscript, err := lbrycrd.ParseClaimUpdateScript(*script)
	if err != nil {
		errors.Prefix("Claim update processing error: ", err)
		return name, claimId, pubkeyscript, err
	}
	claim, err := lbrycrd.DecodeClaimValue(name, value)
	if claim != nil {
		//log.Debug("ClaimUpdate ", name, " ClaimId ", claimId)
	}
	return name, claimId, pubkeyscript, err
}

func GetAddressFromClaimASM(asm string) string {
	sections := strings.Split(asm, " ")
	address, err := lbrycrd.GetAddressFromP2PKH(sections[7])
	if err != nil {
		panic(err)
	}

	return address
}
