package lbrycrd

import "github.com/btcsuite/btcd/btcjson"

// ClaimNameResult models the data from the claimtrie of lbrycrd.
type ClaimNameResult struct {
	Name   string        `json:"name"`
	Claims []ClaimResult `json:"claims,omitempty"`
}

// ClaimResult models the static data of a claim in the claimtrie
type ClaimResult struct {
	ClaimID  string  `json:"claimId"`
	TxID     string  `json:"txId"`
	Sequence uint64  `json:"n"`
	Amount   float64 `json:"amount"`
	Height   uint64  `json:"height"`
	Value    string  `json:"effectiveValue"`
}

// GetBlockHeaderResponse models the data from the getblockheader command when
// the verbose flag is set.  When the verbose flag is not set, getblockheader
// returns a hex-encoded string.
type GetBlockHeaderResponse struct {
	Hash              string  `json:"hash"`
	Confirmations     uint64  `json:"confirmations"`
	Height            int32   `json:"height"`
	Version           int32   `json:"version"`
	VersionHex        string  `json:"versionHex"`
	MerkleRoot        string  `json:"merkleroot"`
	NameClaimRoot     string  `json:"nameclaimroot"`
	Time              int64   `json:"time"`
	MedianTime        int64   `json:"mediantime"`
	Nonce             uint64  `json:"nonce"`
	Bits              string  `json:"bits"`
	Difficulty        float64 `json:"difficulty"`
	ChainWork         string  `json:"chainwork"`
	NTx               int32   `json:"nTx"`
	PreviousBlockHash string  `json:"previousblockhash,omitempty"`
	NextBlockHash     string  `json:"nextblockhash,omitempty"`
}

// GetBlockResponse models the data from the getblock command when the
// verbose flag is set.  When the verbose flag is not set, getblock returns a
// hex-encoded string.
type GetBlockResponse struct {
	Hash              string   `json:"hash"`
	Confirmations     uint64   `json:"confirmations"`
	StrippedSize      int32    `json:"strippedsize"`
	Size              int32    `json:"size"`
	Weight            int32    `json:"weight"`
	Height            int64    `json:"height"`
	Version           int32    `json:"version"`
	VersionHex        string   `json:"versionHex"`
	MerkleRoot        string   `json:"merkleroot"`
	NameClaimRoot     string   `json:"nameclaimroot"`
	Tx                []string `json:"tx"`
	Time              int64    `json:"time"`
	MedianTime        int64    `json:"mediantime"`
	Nonce             uint64   `json:"nonce"`
	Bits              string   `json:"bits"`
	Difficulty        float64  `json:"difficulty"`
	ChainWork         string   `json:"chainwork"`
	NTx               int32    `json:"nTx"`
	PreviousBlockHash string   `json:"previousblockhash"`
	NextBlockHash     string   `json:"nextblockhash,omitempty"`
}

// GetBlockVerboseResponse models the data from the getblock command when the
// verbose flag is set.  When the verbose flag is not set, getblock returns a
// hex-encoded string.
type GetBlockVerboseResponse struct {
	Hash              string        `json:"hash"`
	Confirmations     int64         `json:"confirmations"`
	StrippedSize      int32         `json:"strippedsize"`
	Size              int32         `json:"size"`
	Weight            int32         `json:"weight"`
	Height            int64         `json:"height"`
	Version           int32         `json:"version"`
	VersionHex        string        `json:"versionHex"`
	MerkleRoot        string        `json:"merkleroot"`
	NameClaimRoot     string        `json:"nameclaimroot"`
	Tx                []TxRawResult `json:"tx,omitempty"`
	Time              int64         `json:"time"`
	MedianTime        int64         `json:"mediantime"`
	Nonce             uint64        `json:"nonce"`
	Bits              string        `json:"bits"`
	Difficulty        float64       `json:"difficulty"`
	ChainWork         string        `json:"chainwork"`
	NTx               int32         `json:"nTx"`
	PreviousBlockHash string        `json:"previousblockhash"`
	NextBlockHash     string        `json:"nextblockhash,omitempty"`
}

// TxRawResult models the data from the getrawtransaction command.
// TxRawResult models the data from the getrawtransaction command.
type TxRawResult struct {
	Txid          string `json:"txid"`
	Hash          string `json:"hash,omitempty"`
	Version       int32  `json:"version"`
	Size          int32  `json:"size,omitempty"`
	Vsize         int32  `json:"vsize,omitempty"`
	Weight        int32  `json:"weight"`
	LockTime      uint64 `json:"locktime"`
	Vin           []Vin  `json:"vin"`
	Vout          []Vout `json:"vout"`
	Hex           string `json:"hex"`
	BlockHash     string `json:"blockhash,omitempty"`
	Confirmations uint64 `json:"confirmations,omitempty"`
	Time          int64  `json:"time,omitempty"`
	Blocktime     int64  `json:"blocktime,omitempty"`
}

// Vout models parts of the tx data.  It is defined separately since both
// getrawtransaction and decoderawtransaction use the same structure.
type Vout struct {
	Value        float64                    `json:"value"`
	N            uint64                     `json:"n"`
	ScriptPubKey btcjson.ScriptPubKeyResult `json:"scriptPubKey"`
}

// Vin models parts of the tx data.  It is defined separately since
// getrawtransaction, decoderawtransaction, and searchrawtransaction use the
// same structure.
type Vin struct {
	Coinbase  string             `json:"coinbase"`
	TxID      string             `json:"txid"`
	Vout      uint64             `json:"vout"`
	ScriptSig *btcjson.ScriptSig `json:"scriptSig"`
	Sequence  uint64             `json:"sequence"`
	Witness   []string           `json:"witness"`
}

// ClaimsForNameResult models the claim list for a name in the claimtrie of lbrycrd.
type ClaimsForNameResult struct {
	NormalizedName       string    `json:"normalizedName"`
	Claims               []Claim   `json:"claims"`
	LastTakeOverHeight   int32     `json:"lastTakeoverHeight"`
	SupportsWithoutClaim []Support `json:"supportsWithoutClaim"`
}

// Claim models the data of a claim both static and dynamic. Used for claimtrie sync.
type Claim struct {
	Name            string    `json:"name,omitempty"`
	Value           string    `json:"value"`
	Address         string    `json:"address"`
	ClaimID         string    `json:"claimId"`
	TxID            string    `json:"txId"`
	N               int32     `json:"n"`
	Height          int32     `json:"height"`
	ValidAtHeight   int32     `json:"validAtHeight"`
	Amount          uint64    `json:"amount"`
	EffectiveAmount uint64    `json:"effectiveAmount"`
	PendingAmount   uint64    `json:"pendingAmount"`
	Supports        []Support `json:"supports,omitempty"`
	Bid             uint64    `json:"bid"`
	Sequence        int       `json:"sequence"`
}

// Support models the support information for a claim in the claimtrie of lbrycrd.
type Support struct {
	Address       string `json:"address"`
	TxID          string `json:"txId"`
	N             int32  `json:"n"`
	Height        int32  `json:"height"`
	ValidAtHeight int32  `json:"validAtHeight"`
	Amount        uint64 `json:"amount"`
}

// GetRawMempoolVerboseResult models the data returned from the getrawmempool
// command when the verbose flag is set.  When the verbose flag is not set,
// getrawmempool returns an array of transaction hashes.
type GetRawMempoolVerboseResult struct {
	Fees struct {
		Base       float64 `json:"base"`
		Modified   float64 `json:"modified"`
		Ancestor   float64 `json:"ancestor"`
		Descendant float64 `json:"descendant"`
	} `json:"fees"`
	Size            int      `json:"size"`
	Fee             float64  `json:"fee"`
	ModifiedFee     float64  `json:"modifiedfee"`
	Time            int64    `json:"time"`
	Height          int32    `json:"height"`
	DescendantCount int32    `json:"descendantcount"`
	DescendantSize  int32    `json:"descendantsize"`
	DescendantFees  uint64   `json:"descendantfees"`
	AncestorCount   int32    `json:"ancestorcount"`
	AncestorSize    int32    `json:"ancestorsize"`
	AncestorFees    int32    `json:"ancestorfees"`
	WtxID           string   `json:"wtxid"`
	Depends         []string `json:"depends"`
	SpentBy         []string `json:"spentby"`
}
