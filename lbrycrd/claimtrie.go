package lbrycrd

import (
	"github.com/btcsuite/btcutil"
)

func Hash160(bytes []byte) []byte {
	hashBytes := btcutil.Hash160(bytes)
	println("Hash160 bytes: ", len(hashBytes))
	return hashBytes
}
