package datastore

import "sync"

// TransactionAddress Caching
type txAddressKey struct {
	txID   uint64
	addrID uint64
}
type tACache map[txAddressKey]bool

var txAddrCache = tACache{}
var tcAddrLock = sync.RWMutex{}

// Output Caching
type outputKey struct {
	txHash string
	vout   uint
}
type oCache map[outputKey]bool

var outputCache = oCache{}
var outputLock = sync.RWMutex{}

func checkTxAddrCache(key txAddressKey) bool {
	tcAddrLock.RLock()
	defer tcAddrLock.RUnlock()
	return txAddrCache[key]
}

func checkOutputCache(key outputKey) bool {
	outputLock.RLock()
	defer outputLock.RUnlock()
	return outputCache[key]
}

func addToTxAddrCache(key txAddressKey) {
	tcAddrLock.Lock()
	defer tcAddrLock.Unlock()
	txAddrCache[key] = true
}

func addToOuputCache(key outputKey) {
	outputLock.Lock()
	defer outputLock.Unlock()
	outputCache[key] = true
}
