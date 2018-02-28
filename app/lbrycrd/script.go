package lbrycrd

import (
	"encoding/binary"
	"encoding/hex"
	log "github.com/sirupsen/logrus"
)

const (
	//lbrycrd				//btcd
	OP_CLAIM_NAME    = 0xb5 //OP_NOP6 		= 181
	OP_SUPPORT_CLAIM = 0xb6 //OP_NOP7 		= 182
	OP_UPDATE_CLAIM  = 0xb7 //OP_NOP8 		= 183
	OP_HASH160       = 0xa9 //OP_HASH160	= 169
	OP_PUSHDATA1     = 0x4c //OP_PUSHDATA1  = 76
	OP_PUSHDATA2     = 0x4d //OP_PUSHDATA2 	= 77
	OP_PUSHDATA4     = 0x4e //OP_PUSHDATA4 	= 78
)

func IsClaimNameScript(script []byte) bool {
	if script != nil && len(script) > 0 {
		return script[0] == OP_CLAIM_NAME
	}
	log.Error("script is nil or length 0!")
	return false
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

func IsPayToHashScript(script []byte) bool {
	if script != nil && len(script) > 0 {
		return script[0] == OP_UPDATE_CLAIM
	}
	return false
}
