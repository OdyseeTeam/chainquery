package lbrycrd

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/lbryio/chainquery/global"

	"github.com/lbryio/lbry.go/errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	log "github.com/sirupsen/logrus"
)

const (
	//lbrycrd				//btcd
	opClaimName    = 0xb5 //OP_NOP6 			= 181
	opSupportClaim = 0xb6 //OP_NOP7 			= 182
	opUpdateClaim  = 0xb7 //OP_NOP8 			= 183
	opDup          = 0x76 //opDup 			= 118
	opChecksig     = 0xac //opChecksig 		= 172
	opEqualverify  = 0x88 //opEqualverify 	= 136
	opHash160      = 0xa9 //opHash160		= 169
	opPushdata1    = 0x4c //opPushdata1  	= 76
	opPushdata2    = 0x4d //opPushdata2 		= 77
	opPushdata4    = 0x4e //opPushdata4 		= 78

	// Types of vOut scripts
	p2SH  = "scripthash" // Pay to Script Hash
	p2PK  = "pubkey"     // Pay to Public Key
	p2PKH = "pubkeyhash" // Pay to Public Key Hash
	// NonStandard is a transaction type, usually used for a claim.
	NonStandard = "nonstandard" // Non Standard - Used for Claims in LBRY

	lbrycrdMainPubkeyPrefix    = byte(85)
	lbrycrdMainScriptPrefix    = byte(122)
	lbrycrdTestnetPubkeyPrefix = byte(111)
	lbrycrdTestnetScriptPrefix = byte(196)
	lbrycrdRegtestPubkeyPrefix = byte(111)
	lbrycrdRegtestScriptPrefix = byte(196)

	lbrycrdMain    = "lbrycrd_main"
	lbrycrdTestnet = "lbrycrd_testnet"
	lbrycrdRegtest = "lbrycrd_regtest"
)

var mainNetParams = chaincfg.Params{
	PubKeyHashAddrID: lbrycrdMainPubkeyPrefix,
	ScriptHashAddrID: lbrycrdMainScriptPrefix,
	PrivateKeyID:     0x1c,
}

var testNetParams = chaincfg.Params{
	PubKeyHashAddrID: lbrycrdTestnetPubkeyPrefix,
	ScriptHashAddrID: lbrycrdTestnetScriptPrefix,
	PrivateKeyID:     0x1c,
}

var regTestNetParams = chaincfg.Params{
	PubKeyHashAddrID: lbrycrdRegtestPubkeyPrefix,
	ScriptHashAddrID: lbrycrdRegtestScriptPrefix,
	PrivateKeyID:     0x1c,
}

var paramsMap = map[string]chaincfg.Params{lbrycrdMain: mainNetParams, lbrycrdTestnet: testNetParams, lbrycrdRegtest: regTestNetParams}

// IsClaimScript return true if the script for the vout contains the right opt codes pertaining to a claim.
func IsClaimScript(script []byte) bool {
	return script[0] == opSupportClaim ||
		script[0] == opClaimName ||
		script[0] == opUpdateClaim
}

// IsClaimNameScript returns true if the script for the vout contains the OP_CLAIM_NAME code.
func IsClaimNameScript(script []byte) bool {
	if len(script) > 0 {
		return script[0] == opClaimName
	}
	log.Error("script is nil or length 0!")
	return false
}

// IsClaimSupportScript returns true if the script for the vout contains the OP_CLAIM_SUPPORT code.
func IsClaimSupportScript(script []byte) bool {
	if len(script) > 0 {
		return script[0] == opSupportClaim
	}
	return false
}

// IsClaimUpdateScript returns true if the script for the vout contains the OP_CLAIM_UPDATE code.
func IsClaimUpdateScript(script []byte) bool {
	if len(script) > 0 {
		return script[0] == opUpdateClaim
	}
	return false
}

// ParseClaimNameScript parses a script for the claim of a name.
func ParseClaimNameScript(script []byte) (name string, value []byte, pubkeyscript []byte, err error) {
	// Already validated by blockchain so can be assumed
	// opClaimName Name Value OP_2DROP OP_DROP pubkeyscript
	nameBytesToRead := int(script[1])
	nameStart := 2
	if nameBytesToRead == opPushdata1 {
		nameBytesToRead = int(script[2])
		nameStart = 3
	} else if nameBytesToRead > opPushdata1 {
		panic(errors.Base("Bytes to read is more than next byte! "))
	}
	nameEnd := nameStart + nameBytesToRead
	name = string(script[nameStart:nameEnd])
	dataPushType := int(script[nameEnd])
	valueBytesToRead := int(script[nameEnd])
	valueStart := nameEnd + 1
	if dataPushType == opPushdata1 {
		valueBytesToRead = int(script[nameEnd+1])
		valueStart = nameEnd + 2
	} else if dataPushType == opPushdata2 {
		valueStart = nameEnd + 3
		valueBytesToRead = int(binary.LittleEndian.Uint16(script[nameEnd+1 : valueStart]))
	} else if dataPushType == opPushdata4 {
		valueStart = nameEnd + 5
		valueBytesToRead = int(binary.LittleEndian.Uint32(script[nameEnd+2 : valueStart]))
	}
	valueEnd := valueStart + valueBytesToRead
	value = script[valueStart:valueEnd]
	pksStart := valueEnd + 2         // +2 to ignore OP_2DROP and OP_DROP
	pubkeyscript = script[pksStart:] //Remainder is always pubkeyscript

	return name, value, pubkeyscript, err
}

// ParseClaimSupportScript parses a script for a support of a claim.
func ParseClaimSupportScript(script []byte) (name string, claimid string, pubkeyscript []byte, err error) {
	// Already validated by blockchain so can be assumed
	// opSupportClaim vchName vchClaimId OP_2DROP OP_DROP pubkeyscript

	//Name
	nameBytesToRead := int(script[1])
	nameStart := 2
	if nameBytesToRead == opPushdata1 {
		nameBytesToRead = int(script[2])
		nameStart = 3
	} else if nameBytesToRead > opPushdata1 {
		panic(errors.Base("Bytes to read is more than next byte! "))
	}
	nameEnd := nameStart + nameBytesToRead
	name = string(script[nameStart:nameEnd])

	//ClaimID
	claimidBytesToRead := int(script[nameEnd])
	claimidStart := nameEnd + 1
	claimidEnd := claimidStart + claimidBytesToRead
	bytes := reverseBytes(script[claimidStart:claimidEnd])
	claimid = hex.EncodeToString(bytes)

	//PubKeyScript
	pksStart := claimidEnd + 2       // +2 to ignore OP_2DROP and OP_DROP
	pubkeyscript = script[pksStart:] //Remainder is always pubkeyscript
	return
}

// ParseClaimUpdateScript parses a script for an update of a claim.
func ParseClaimUpdateScript(script []byte) (name string, claimid string, value []byte, pubkeyscript []byte, err error) {
	// opUpdateClaim Name ClaimID Value OP_2DROP OP_2DROP pubkeyscript

	//Name
	nameBytesToRead := int(script[1])
	nameStart := 2
	if nameBytesToRead == opPushdata1 {
		nameBytesToRead = int(script[2])
		nameStart = 3
	} else if nameBytesToRead > opPushdata1 {
		panic(errors.Base("Bytes to read is more than next byte! "))
	}
	nameEnd := nameStart + nameBytesToRead
	name = string(script[nameStart:nameEnd])

	//ClaimID
	claimidBytesToRead := int(script[nameEnd])
	claimidStart := nameEnd + 1
	claimidEnd := claimidStart + claimidBytesToRead
	bytes := reverseBytes(script[claimidStart:claimidEnd])
	claimid = hex.EncodeToString(bytes)

	//Value
	dataPushType := int(script[claimidEnd])
	valueBytesToRead := int(script[claimidEnd])
	valueStart := claimidEnd + 1
	if dataPushType == opPushdata1 {
		valueBytesToRead = int(script[claimidEnd+1])
		valueStart = claimidEnd + 2
	} else if dataPushType == opPushdata2 {
		valueStart = claimidEnd + 3
		valueBytesToRead = int(binary.LittleEndian.Uint16(script[claimidEnd+1 : valueStart]))
	} else if dataPushType == opPushdata4 {
		valueStart = claimidEnd + 5
		valueBytesToRead = int(binary.LittleEndian.Uint32(script[claimidEnd+2 : valueStart]))
	}
	valueEnd := valueStart + valueBytesToRead
	value = script[valueStart:valueEnd]

	//PublicKeyScript
	pksStart := valueEnd + 2         // +2 to ignore OP_2DROP and OP_DROP
	pubkeyscript = script[pksStart:] //Remainder is always pubkeyscript

	return name, claimid, value, pubkeyscript, err
}

// GetPubKeyScriptFromClaimPKS gets the public key script at the end of a claim script.
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

// GetAddressFromPublicKeyScript returns the address associated with a public key script.
func GetAddressFromPublicKeyScript(script []byte) (address string) {
	spkType := getPublicKeyScriptType(script)
	var err error
	switch spkType {
	case p2PK:
		// <pubkey> opChecksig
		//log.Debug("sig p2PK ", hex.EncodeToString(script[0:len(script)-1]))
		address, err = getAddressFromP2PK(hex.EncodeToString(script[0 : len(script)-1]))
	case p2PKH:
		// opDup opHash160 <bytes2read> <PubKeyHash> opEqualverify opChecksig
		//log.Debug("sig p2PKH ", hex.EncodeToString(script[3:len(script)-2]))
		address, err = getAddressFromP2PKH(hex.EncodeToString(script[3 : len(script)-2]))
	case p2SH:
		// opHash160 <bytes2read> <Hash160(redeemScript)> OP_EQUAL
		//log.Debug("sig p2SH ", hex.EncodeToString(script[2:len(script)-1]))
		address, err = getAddressFromP2SH(hex.EncodeToString(script[2 : len(script)-1]))
	case NonStandard:
		address = "UNKNOWN"
	}

	if err != nil {
		panic(err)
	}

	return address
}

func getPublicKeyScriptType(script []byte) string {
	if isPayToPublicKey(script) {
		return p2PK
	} else if isPayToPublicKeyHashScript(script) {
		return p2PKH
	} else if isPayToScriptHashScript(script) {
		return p2SH
	}
	return NonStandard
}

func isPayToScriptHashScript(script []byte) bool {
	if len(script) > 0 {
		return script[0] == opUpdateClaim
	}
	return false
}

func isPayToPublicKey(script []byte) bool {
	return script[len(script)-1] == opChecksig &&
		script[len(script)-2] == opEqualverify &&
		script[0] != opDup

}

func isPayToPublicKeyHashScript(script []byte) bool {
	return len(script) > 0 &&
		script[0] == opDup &&
		script[1] == opHash160

}

func getAddressFromP2PK(hexstring string) (string, error) {
	hexstringBytes, err := hex.DecodeString(hexstring)
	if err != nil {
		return "", err
	}
	chainParams, ok := paramsMap[global.BlockChainName]
	if !ok {
		return "", errors.Err("unknown chain name %s", global.BlockChainName)
	}
	addr, err := btcutil.NewAddressPubKey(hexstringBytes, &chainParams)
	if err != nil {
		return "", err
	}
	address := addr.EncodeAddress()

	return address, nil
}

func getAddressFromP2PKH(hexstring string) (string, error) {
	hexstringBytes, err := hex.DecodeString(hexstring)
	if err != nil {
		return "", err
	}
	chainParams, ok := paramsMap[global.BlockChainName]
	if !ok {
		return "", errors.Err("unknown chain name %s", global.BlockChainName)
	}
	addr, err := btcutil.NewAddressPubKeyHash(hexstringBytes, &chainParams)
	if err != nil {
		return "", err
	}
	address := addr.EncodeAddress()
	return address, nil

}

func getAddressFromP2SH(hexstring string) (string, error) {
	hexstringBytes, err := hex.DecodeString(hexstring)
	if err != nil {
		return "", err
	}

	chainParams, ok := paramsMap[global.BlockChainName]
	if !ok {
		return "", errors.Err("unknown chain name %s", global.BlockChainName)
	}
	addr, err := btcutil.NewAddressScriptHashFromHash(hexstringBytes, &chainParams)
	if err != nil {
		return "", err
	}
	address := addr.EncodeAddress()
	return address, nil
}

// rev reverses a byte slice. useful for switching endian-ness
func reverseBytes(b []byte) []byte {
	r := make([]byte, len(b))
	for left, right := 0, len(b)-1; left < right; left, right = left+1, right-1 {
		r[left], r[right] = b[right], b[left]
	}
	return r
}
