package lbrycrd

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"

	"github.com/lbryio/chainquery/util"
	"golang.org/x/crypto/ripemd160"
)

func ClaimIDFromOutpoint(txid string, nout int) (string, error) {
	txidBytes, err := hex.DecodeString(txid)
	if err != nil {
		return "", err
	}
	txidBytes = util.ReverseBytes(txidBytes)
	noutBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(noutBytes, uint32(nout))
	txidBytes = append(txidBytes, noutBytes...)
	s := sha256.New()
	_, err = s.Write(txidBytes)
	if err != nil {
		return "", err
	}
	r := ripemd160.New()
	_, err = r.Write(s.Sum(nil))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(util.ReverseBytes(r.Sum(nil))), nil
}
