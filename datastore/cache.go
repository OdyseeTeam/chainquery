package datastore

import "sync"

// TransactionAddress Caching
type txAddressKey struct {
	txID   uint64
	addrID uint64
}
type TACache map[txAddressKey]bool

var txAddrCache = TACache{}
var tcAddrLock = sync.RWMutex{}

// Output Caching
type outputKey struct {
	txHash string
	vout   uint
}
type OCache map[outputKey]bool

var outputCache = OCache{}
var outputLock = sync.RWMutex{}

func CheckTxAddrCache(key txAddressKey) bool {
	tcAddrLock.RLock()
	defer tcAddrLock.RUnlock()
	return txAddrCache[key]
}

func CheckOutputCache(key outputKey) bool {
	outputLock.RLock()
	defer outputLock.RUnlock()
	return outputCache[key]
}

func AddToTxAddrCache(key txAddressKey) {
	tcAddrLock.Lock()
	defer tcAddrLock.Unlock()
	txAddrCache[key] = true
}

func AddToOuputCache(key outputKey) {
	outputLock.Lock()
	defer outputLock.Unlock()
	outputCache[key] = true
}
