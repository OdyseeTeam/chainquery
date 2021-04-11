package lbrycrd

import (
	"encoding/binary"
	"encoding/hex"

	"github.com/golang/protobuf/proto"

	"github.com/lbryio/chainquery/global"
	"github.com/lbryio/chainquery/util"

	"github.com/lbryio/lbry.go/extras/errors"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	pb "github.com/lbryio/types/v2/go"
	log "github.com/sirupsen/logrus"
)

const (
	//lbrycrd				//btcd
	opClaimName    = 0xb5 //OP_NOP6 			= 181
	opSupportClaim = 0xb6 //OP_NOP7 			= 182
	opUpdateClaim  = 0xb7 //OP_NOP8 			= 183
	opReturn       = 0x6a //OP_RETURN       	= 106
	SOMETHING      = 0x17 // = 23
	purchase       = 0x50 //PURCHASE = 80
	opDup          = 0x76 //opDup 				= 118
	opChecksig     = 0xac //opChecksig 			= 172
	opEqualverify  = 0x88 //opEqualverify 		= 136
	opHash160      = 0xa9 //opHash160			= 169
	opPushdata1    = 0x4c //opPushdata1  		= 76
	opPushdata2    = 0x4d //opPushdata2 		= 77
	opPushdata4    = 0x4e //opPushdata4 		= 78

	// Types of vOut scripts
	p2SH   = "scripthash"            // Pay to Script Hash
	p2PK   = "pubkey"                // Pay to Public Key
	p2PKH  = "pubkeyhash"            // Pay to Public Key Hash
	p2WPKH = "witness_v0_keyhash"    //Segwit Pub Key Hash
	p2WSH  = "witness_v0_scripthash" //Segwit Script Hash
	// NonStandard is a transaction type, usually used for a claim.
	NonStandard = "nonstandard"
	// NullData Transaction type related to segwit outputs
	NullData = "nulldata"

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
	Bech32HRPSegwit:  "lbc",
}

var testNetParams = chaincfg.Params{
	PubKeyHashAddrID: lbrycrdTestnetPubkeyPrefix,
	ScriptHashAddrID: lbrycrdTestnetScriptPrefix,
	PrivateKeyID:     0x1c,
	Bech32HRPSegwit:  "tlbc",
}

var regTestNetParams = chaincfg.Params{
	PubKeyHashAddrID: lbrycrdRegtestPubkeyPrefix,
	ScriptHashAddrID: lbrycrdRegtestScriptPrefix,
	PrivateKeyID:     0x1c,
	Bech32HRPSegwit:  "rlbc",
}

var paramsMap = map[string]chaincfg.Params{lbrycrdMain: mainNetParams, lbrycrdTestnet: testNetParams, lbrycrdRegtest: regTestNetParams}

//GetChainParams returns the currently set blockchain name as the chain parameters. Set in the config.
func GetChainParams() (*chaincfg.Params, error) {
	chainParams, ok := paramsMap[global.BlockChainName]
	if !ok {
		return nil, errors.Err("unknown chain name %s", global.BlockChainName)
	}

	return &chainParams, nil
}

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

// IsPurchaseScript returns true if the script for the vout contains the OP_RETURN + 'P' byte identifier for a purchase
func IsPurchaseScript(script []byte) bool {
	if len(script) > 2 {
		return script[0] == opReturn && script[2] == purchase
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
		err = errors.Err("Bytes to read is more than next byte! ")
		return
	}
	nameEnd := nameStart + nameBytesToRead
	name = string(script[nameStart:nameEnd])

	//ClaimID
	claimidBytesToRead := int(script[nameEnd])
	claimidStart := nameEnd + 1
	claimidEnd := claimidStart + claimidBytesToRead
	bytes := util.ReverseBytes(script[claimidStart:claimidEnd])
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
		err = errors.Err("ParseClaimUpdateScript: Bytes to read is more than next byte! ")
		return
	}
	nameEnd := nameStart + nameBytesToRead
	name = string(script[nameStart:nameEnd])

	//ClaimID
	claimidBytesToRead := int(script[nameEnd])
	claimidStart := nameEnd + 1
	claimidEnd := claimidStart + claimidBytesToRead
	bytes := util.ReverseBytes(script[claimidStart:claimidEnd])
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

//ErrNotClaimScript is a base error for when a script cannot be parsed as a claim script.
var ErrNotClaimScript = errors.Base("Script is not a claim script!")

// GetPubKeyScriptFromClaimPKS gets the public key script at the end of a claim script.
func GetPubKeyScriptFromClaimPKS(script []byte) (pubkeyscript []byte, err error) {
	if IsClaimScript(script) {
		if IsClaimNameScript(script) {
			_, _, pubkeyscript, err = ParseClaimNameScript(script)
			if err != nil {
				return nil, errors.Err(err)
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
		err = ErrNotClaimScript
	}
	return
}

// GetAddressFromPublicKeyScript returns the address associated with a public key script.
func GetAddressFromPublicKeyScript(script []byte) (address string) {
	chainParams, err := GetChainParams()
	if err != nil {
		return ""
	}
	_, BTCAddress, _, err := txscript.ExtractPkScriptAddrs(script, chainParams)
	if len(BTCAddress) < 1 {
		return ""
	}
	address = BTCAddress[0].EncodeAddress()

	return address
}

func getPublicKeyScriptType(script []byte) string {
	if isPayToPublicKey(script) {
		return p2PK
	} else if isPayToPublicKeyHashScript(script) {
		return p2PKH
	} else if isPayToScriptHashScript(script) {
		return p2SH
	} else if txscript.IsPayToWitnessPubKeyHash(script) {
		return p2WPKH
	} else if txscript.IsPayToWitnessScriptHash(script) {
		return p2WSH
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
	chainParams, err := GetChainParams()
	if err != nil {
		return "", errors.Err(err)
	}
	addr, err := btcutil.NewAddressPubKey(hexstringBytes, chainParams)
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
	chainParams, err := GetChainParams()
	if err != nil {
		return "", errors.Err(err)
	}
	addr, err := btcutil.NewAddressPubKeyHash(hexstringBytes, chainParams)
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

	chainParams, err := GetChainParams()
	if err != nil {
		return "", errors.Err(err)
	}
	addr, err := btcutil.NewAddressScriptHashFromHash(hexstringBytes, chainParams)
	if err != nil {
		return "", err
	}
	address := addr.EncodeAddress()
	return address, nil
}

func getAddressFromP2WPKH(hexstring string) (string, error) {
	witnessProgram, err := hex.DecodeString(hexstring)
	if err != nil {
		return "", err
	}

	chainParams, err := GetChainParams()
	if err != nil {
		return "", errors.Err(err)
	}
	addr, err := btcutil.NewAddressWitnessPubKeyHash(witnessProgram, chainParams)
	if err != nil {
		return "", err
	}
	address := addr.EncodeAddress()
	return address, nil
}

func getAddressFromP2WSH(hexstring string) (string, error) {
	witnessProgram, err := hex.DecodeString(hexstring)
	if err != nil {
		return "", err
	}

	chainParams, err := GetChainParams()
	if err != nil {
		return "", errors.Err(err)
	}
	addr, err := btcutil.NewAddressWitnessScriptHash(witnessProgram, chainParams)
	if err != nil {
		return "", err
	}
	address := addr.EncodeAddress()
	return address, nil
}

func parseDataScript(script []byte) ([]byte, error) {
	// OP_RETURN (bytes) DATA
	if len(script) <= 1 {
		return nil, errors.Err("there is no script to parse")
	}
	if script[0] != opReturn {
		return nil, errors.Err("the first byte of script must be an OP_RETURN to quality as un-spendable data")
	}
	dataBytesToRead := int(script[1])
	if (len(script) - dataBytesToRead - 2) != 0 {
		return nil, errors.Err("supposed to have %d bytes to read but the script is %d bytes", dataBytesToRead, len(script))
	}
	return script[2:], nil
}

func ParsePurchaseScript(script []byte) (*pb.Purchase, error) {
	data, err := parseDataScript(script)
	if err != nil {
		return nil, err
	}
	if data[0] != purchase {
		return nil, errors.Err("the first byte must be 'P'(0x50) to be a purchase script")
	}
	purchase := &pb.Purchase{}
	err = proto.Unmarshal(data[1:], purchase)
	if err != nil {
		return nil, errors.Err(err)
	}
	return purchase, nil
}
