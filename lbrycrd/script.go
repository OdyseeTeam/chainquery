package lbrycrd

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/lbryio/lbry.go/errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	log "github.com/sirupsen/logrus"
)

const (
	//lbrycrd				//btcd
	OP_CLAIM_NAME    = 0xb5 //OP_NOP6 			= 181
	OP_SUPPORT_CLAIM = 0xb6 //OP_NOP7 			= 182
	OP_UPDATE_CLAIM  = 0xb7 //OP_NOP8 			= 183
	OP_DUP           = 0x76 //OP_DUP 			= 118
	OP_CHECKSIG      = 0xac //OP_CHECKSIG 		= 172
	OP_EQUALVERIFY   = 0x88 //OP_EQUALVERIFY 	= 136
	OP_HASH160       = 0xa9 //OP_HASH160		= 169
	OP_PUSHDATA1     = 0x4c //OP_PUSHDATA1  	= 76
	OP_PUSHDATA2     = 0x4d //OP_PUSHDATA2 		= 77
	OP_PUSHDATA4     = 0x4e //OP_PUSHDATA4 		= 78

	// Types of vOut scripts
	P2SH         = "scripthash"  // Pay to Script Hash
	P2PK         = "pubkey"      // Pay to Public Key
	P2PKH        = "pubkeyhash"  // Pay to Public Key Hash
	NON_STANDARD = "nonstandard" // Non Standard - Used for Claims in LBRY
)

var MainNetParams = chaincfg.Params{
	PubKeyHashAddrID: 0x55,
	ScriptHashAddrID: 0x7a,
	PrivateKeyID:     0x1c,
}

func IsClaimNameScript(script []byte) bool {
	if script != nil && len(script) > 0 {
		return script[0] == OP_CLAIM_NAME
	}
	log.Error("script is nil or length 0!")
	return false
}

func IsClaimScript(script []byte) bool {
	return script[0] == OP_SUPPORT_CLAIM ||
		script[0] == OP_CLAIM_NAME ||
		script[0] == OP_UPDATE_CLAIM
}

func ParseClaimNameScript(script []byte) (name string, value []byte, pubkeyscript []byte, err error) {
	// Already validated by blockchain so can be assumed
	// OP_CLAIM_NAME Name Value OP_2DROP OP_DROP pubkeyscript
	nameBytesToRead := int(script[1])
	if nameBytesToRead < OP_PUSHDATA1 {
		nameStart := 2
		nameEnd := nameStart + nameBytesToRead
		name = string(script[nameStart:nameEnd])
		dataPushType := int(script[nameEnd])
		valueBytesToRead := int(script[nameEnd])
		valueStart := nameEnd + 1
		if dataPushType == OP_PUSHDATA1 {
			valueBytesToRead = int(script[nameEnd+1])
			valueStart = nameEnd + 2
		} else if dataPushType == OP_PUSHDATA2 {
			valueStart = nameEnd + 3
			valueBytesToRead = int(binary.LittleEndian.Uint16(script[nameEnd+1 : valueStart]))
		} else if dataPushType == OP_PUSHDATA4 {
			valueStart = nameEnd + 5
			valueBytesToRead = int(binary.LittleEndian.Uint32(script[nameEnd+2 : valueStart]))
		}
		valueEnd := valueStart + valueBytesToRead
		value = script[valueStart:valueEnd]
		pksStart := valueEnd + 2         // +2 to ignore OP_2DROP and OP_DROP
		pubkeyscript = script[pksStart:] //Remainder is always pubkeyscript
	}

	return name, value, pubkeyscript, err
}

func IsClaimSupportScript(script []byte) bool {
	if script != nil && len(script) > 0 {
		return script[0] == OP_SUPPORT_CLAIM
	}
	return false
}

func ParseClaimSupportScript(script []byte) (name string, claimid string, pubkeyscript []byte, err error) {
	// Already validated by blockchain so can be assumed
	// OP_SUPPORT_CLAIM vchName vchClaimId OP_2DROP OP_DROP pubkeyscript
	nameBytesToRead := int(script[1])
	nameStart := 2
	nameEnd := nameStart + nameBytesToRead
	name = string(script[nameStart:nameEnd])
	valueBytesToRead := int(script[nameEnd])
	valueStart := nameEnd + 1
	valueEnd := valueStart + valueBytesToRead
	claimid = hex.EncodeToString(script[valueStart:valueEnd])
	pksStart := valueEnd + 2         // +2 to ignore OP_2DROP and OP_DROP
	pubkeyscript = script[pksStart:] //Remainder is always pubkeyscript
	return
}

func IsClaimUpdateScript(script []byte) bool {
	if script != nil && len(script) > 0 {
		return script[0] == OP_UPDATE_CLAIM
	}
	return false
}

func ParseClaimUpdateScript(script []byte) (name string, claimid string, value []byte, pubkeyscript []byte, err error) {
	// OP_UPDATE_CLAIM Name ClaimId Value OP_2DROP OP_2DROP pubkeyscript
	nameBytesToRead := int(script[1])
	if nameBytesToRead < OP_PUSHDATA1 {
		//Name
		nameStart := 2
		nameEnd := nameStart + nameBytesToRead
		name = string(script[nameStart:nameEnd])

		//ClaimId
		claimidBytesToRead := int(script[nameEnd])
		claimidStart := nameEnd + 1
		claimidEnd := claimidStart + claimidBytesToRead
		claimid = hex.EncodeToString(script[claimidStart:claimidEnd])

		//Value
		dataPushType := int(script[claimidEnd])
		valueBytesToRead := int(script[claimidEnd])
		valueStart := claimidEnd + 1
		if dataPushType == OP_PUSHDATA1 {
			valueBytesToRead = int(script[claimidEnd+1])
			valueStart = claimidEnd + 2
		} else if dataPushType == OP_PUSHDATA2 {
			valueStart = claimidEnd + 3
			valueBytesToRead = int(binary.LittleEndian.Uint16(script[claimidEnd+1 : valueStart]))
		} else if dataPushType == OP_PUSHDATA4 {
			valueStart = claimidEnd + 5
			valueBytesToRead = int(binary.LittleEndian.Uint32(script[claimidEnd+2 : valueStart]))
		}
		valueEnd := valueStart + valueBytesToRead
		value = script[valueStart:valueEnd]

		//PublicKeyScript
		pksStart := valueEnd + 2         // +2 to ignore OP_2DROP and OP_DROP
		pubkeyscript = script[pksStart:] //Remainder is always pubkeyscript
	}

	return name, claimid, value, pubkeyscript, err
}

func GetPubKeyScriptFromClaimPKS(script []byte) (pubkeyscript []byte, err error) {
	if IsClaimScript(script) {
		if IsClaimNameScript(script) {
			_, _, pubkeyscript, err = ParseClaimNameScript(script)
			if err != nil {
				return
			}
			return pubkeyscript, nil
		} else if IsClaimUpdateScript(script) {
			_, _, _, pubkeyscript, err = ParseClaimUpdateScript(script)
			if err != nil {
				return
			}
			return
		} else if IsClaimSupportScript(script) {
			_, _, pubkeyscript, err = ParseClaimSupportScript(script)
			if err != nil {
				return
			}
			return
		}
	} else {
		err = errors.Base("Script is not a claim script!")
	}
	return
}

func GetAddressFromPublicKeyScript(script []byte) (address string) {
	spkType := getPublicKeyScriptType(script)
	var err error
	switch spkType {
	case P2PK:
		// <pubkey> OP_CHECKSIG
		//log.Debug("sig P2PK ", hex.EncodeToString(script[0:len(script)-1]))
		address, err = GetAddressFromP2PK(hex.EncodeToString(script[0 : len(script)-1]))
	case P2PKH:
		// OP_DUP OP_HASH160 <bytes2read> <PubKeyHash> OP_EQUALVERIFY OP_CHECKSIG
		//log.Debug("sig P2PKH ", hex.EncodeToString(script[3:len(script)-2]))
		address, err = GetAddressFromP2PKH(hex.EncodeToString(script[3 : len(script)-2]))
	case P2SH:
		// OP_HASH160 <bytes2read> <Hash160(redeemScript)> OP_EQUAL
		//log.Debug("sig P2SH ", hex.EncodeToString(script[2:len(script)-1]))
		address, err = GetAddressFromP2SH(hex.EncodeToString(script[2 : len(script)-1]))
	case NON_STANDARD:
		address = "UNKNOWN"
	}

	if err != nil {
		panic(err)
	}

	return address
}

func getPublicKeyScriptType(script []byte) string {
	if isPayToPublicKey(script) {
		return P2PK
	} else if isPayToPublicKeyHashScript(script) {
		return P2PKH
	} else if isPayToScriptHashScript(script) {
		return P2SH
	}
	return NON_STANDARD
}

func isPayToScriptHashScript(script []byte) bool {
	if script != nil && len(script) > 0 {
		return script[0] == OP_UPDATE_CLAIM
	}
	return false
}

func isPayToPublicKey(script []byte) bool {
	return script[len(script)-1] == OP_CHECKSIG &&
		script[len(script)-2] == OP_EQUALVERIFY &&
		script[0] != OP_DUP

}

func isPayToPublicKeyHashScript(script []byte) bool {
	return len(script) > 0 &&
		script[0] == OP_DUP &&
		script[1] == OP_HASH160

}

func GetAddressFromP2PK(hexstring string) (string, error) {
	hexstringBytes, err := hex.DecodeString(hexstring)
	if err != nil {
		return "", err
	}
	addr, err := btcutil.NewAddressPubKey(hexstringBytes, &MainNetParams)
	if err != nil {
		return "", err
	}
	address := addr.EncodeAddress()

	return address, nil
}

func GetAddressFromP2PKH(hexstring string) (string, error) {
	hexstringBytes, err := hex.DecodeString(hexstring)
	if err != nil {
		return "", err
	}
	addr, err := btcutil.NewAddressPubKeyHash(hexstringBytes, &MainNetParams)
	if err != nil {
		return "", err
	}
	address := addr.EncodeAddress()
	return address, nil

}

func GetAddressFromP2SH(hexstring string) (string, error) {
	hexstringBytes, err := hex.DecodeString(hexstring)
	if err != nil {
		return "", err
	}
	addr, err := btcutil.NewAddressScriptHashFromHash(hexstringBytes, &MainNetParams)
	if err != nil {
		return "", err
	}
	address := addr.EncodeAddress()
	return address, nil
}
