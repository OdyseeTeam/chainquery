package lbrycrd

import (
	"testing"
)

type HashAddressPair struct {
	hash    string
	address string
}

var P2PKHPairs = []HashAddressPair{
	{"5f7a5a5aab24884b74639e221388e443f1a0a5ef", "bMS7TgmB7CUNB7FsimV2wi27YUNSpTNdSo"},
	{"36c63c9af872095dc7bc5a6adab80b54f7a12e3c", "bHitfKVqDQH8hwsFcSWKNGa7MMoMJf4Cm3"},
	{"e7928d0fdff4f46473e1c83b2721873051d403e9", "bZqiM61b6NBQ3uv6gfKLPHWiNiDyd128Jd"},
	{"244b19c5f733ccf596876f9328c016f24bcd9478", "bG3AzDtxRxFZ3Nv1CYHmvRiZ7voQndauhy"},
	{"1be1ec470deb59fc69850cf05e787f87175a8244", "bFGhap5jZYbnz5UQUHrkGmTh4NiRvjUE4f"},
}
var P2PKPairs = []HashAddressPair{
	{"024ca653fc094c95aa409430caf2eee08fa6e5fbbe78431e0ec9e7cd80193d98f9", "bZi1WEjGtsdAwuZTnNNTCAZLxhHkiHec4m"},
	{"044ca653fc094c95aa409430caf2eee08fa6e5fbbe78431e0ec9e7cd80193d98f991b8e88792b46d622d128b146e7aca49fbbf858f1e7e452b0e7ae556d5b4556e", "bRpUYMFSHGASCEAW22cVCf4iFeKB2BHEq9"}}
var P2SHPairs = []HashAddressPair{
	{"a6e68448580140c4861a920c7d5140065d45e14b", "rMT5Sg8SyFP3ax2PRaweRCRZoMeYw4znEi"},
	{"6c4aab30dc6cd9c07c40a598f2ee5f41bea3b750", "rG7BZ3EmPMLcggYYkRTveXv8pqedWPDG7p"},
	{"599885176d5d868c72f7327f573f37b4f91d0fa6", "rEQKyb7nd7UUGyEEn5xRkk1fgXdTCf2ZCg"},
	{"20b7bd1bc21a55cbf6b2d554eb48b669eb6d1263", "r9DarmxyPjWkF7ocyxMzaNZN3a9gJvNTZJ"},
}

func TestGetAddressFromP2PKH(t *testing.T) {
	for _, pair := range P2PKHPairs {
		good := pair.address
		result, err := GetAddressFromP2PKH(pair.hash)
		if err != nil {
			t.Error(err)
		} else if result != good {
			t.Error("Failure - Expected ", good, " got ", result)
		}
	}
}

func TestGetAddressFromP2PK(t *testing.T) {
	for _, pair := range P2PKPairs {
		good := pair.address
		result, err := GetAddressFromP2PK(pair.hash)
		if err != nil {
			t.Error(err)
		} else if result != good {
			t.Error("Failure - Expected ", good, " got ", result)
		}
	}
}

func TestGetAddressFromP2SH(t *testing.T) {
	for _, pair := range P2SHPairs {
		good := pair.address
		result, err := GetAddressFromP2SH(pair.hash)
		if err != nil {
			t.Error(err)
		} else if result != good {
			t.Error("Failure - Expected ", good, " got ", result)
		}
	}
}
