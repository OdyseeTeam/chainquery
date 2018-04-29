package lbrycrd

// ClaimNameResult models the data from the claimtrie of lbrycrd.
type ClaimNameResult struct {
	Name   string        `json:"name"`
	Claims []ClaimResult `json:"claims,omitempty"`
}

// ClaimResult models the static data of a claim in the claimtrie
type ClaimResult struct {
	ClaimID  string  `json:"claimId"`
	TxID     string  `json:"txid"`
	Sequence uint64  `json:"n"`
	Amount   float64 `json:"amount"`
	Height   uint64  `json:"height"`
	Value    string  `json:"value"`
}

// GetBlockHeaderResponse models the data from the getblockheader command when
// the verbose flag is set.  When the verbose flag is not set, getblockheader
// returns a hex-encoded string.
type GetBlockHeaderResponse struct {
	Hash          string  `json:"hash"`
	Confirmations uint64  `json:"confirmations"`
	Height        int32   `json:"height"`
	Version       int32   `json:"version"`
	VersionHex    string  `json:"versionHex"`
	MerkleRoot    string  `json:"merkleroot"`
	Time          int64   `json:"time"`
	Nonce         uint64  `json:"nonce"`
	Bits          string  `json:"bits"`
	Difficulty    float64 `json:"difficulty"`
	PreviousHash  string  `json:"previousblockhash,omitempty"`
	NextHash      string  `json:"nextblockhash,omitempty"`
}

// GetBlockResponse models the data from the getblock command when the
// verbose flag is set.  When the verbose flag is not set, getblock returns a
// hex-encoded string.
type GetBlockResponse struct {
	Hash          string   `json:"hash"`
	Confirmations uint64   `json:"confirmations"`
	StrippedSize  int32    `json:"strippedsize"`
	Size          int32    `json:"size"`
	Weight        int32    `json:"weight"`
	Height        int64    `json:"height"`
	Version       int32    `json:"version"`
	VersionHex    string   `json:"versionHex"`
	MerkleRoot    string   `json:"merkleroot"`
	NameClaimRoot string   `json:"nameclaimroot"`
	Tx            []string `json:"tx"`
	Time          int64    `json:"time"`
	Nonce         uint64   `json:"nonce"`
	Bits          string   `json:"bits"`
	Difficulty    float64  `json:"difficulty"`
	PreviousHash  string   `json:"previousblockhash"`
	NextHash      string   `json:"nextblockhash,omitempty"`
	ChainWork     string   `json:"chainwork"`
}

// TxRawResult models the data from the getrawtransaction command.
type TxRawResult struct {
	Hex           string `json:"hex"`
	Txid          string `json:"txid"`
	Hash          string `json:"hash,omitempty"`
	Size          int32  `json:"size,omitempty"`
	Vsize         int32  `json:"vsize,omitempty"`
	Version       int32  `json:"version"`
	LockTime      uint64 `json:"locktime"`
	Vin           []Vin  `json:"vin"`
	Vout          []Vout `json:"vout"`
	BlockHash     string `json:"blockhash,omitempty"`
	Confirmations uint64 `json:"confirmations,omitempty"`
	Time          int64  `json:"time,omitempty"`
	Blocktime     int64  `json:"blocktime,omitempty"`
}

// Vout models parts of the tx data.  It is defined separately since both
// getrawtransaction and decoderawtransaction use the same structure.
type Vout struct {
	Value        float64            `json:"value"`
	N            uint64             `json:"n"`
	ScriptPubKey ScriptPubKeyResult `json:"scriptPubKey"`
}

// Vin models parts of the tx data.  It is defined separately since
// getrawtransaction, decoderawtransaction, and searchrawtransaction use the
// same structure.
type Vin struct {
	Coinbase  string     `json:"coinbase"`
	TxID      string     `json:"txid"`
	Vout      uint64     `json:"vout"`
	ScriptSig *ScriptSig `json:"scriptSig"`
	Sequence  uint64     `json:"sequence"`
}

// ScriptPubKeyResult models the scriptPubKey data of a tx script.  It is
// defined separately since it is used by multiple commands.
type ScriptPubKeyResult struct {
	Asm       string   `json:"asm"`
	Hex       string   `json:"hex,omitempty"`
	ReqSigs   int32    `json:"reqSigs,omitempty"`
	Type      string   `json:"type"`
	Addresses []string `json:"addresses,omitempty"`
}

// ScriptSig models a signature script.  It is defined separately since it only
// applies to non-coinbase.  Therefore the field in the Vin structure needs
// to be a pointer.
type ScriptSig struct {
	Asm string `json:"asm"`
	Hex string `json:"hex"`
}

// ClaimsForNameResult models the claim list for a name in the claimtrie of lbrycrd.
type ClaimsForNameResult struct {
	LastTakeOverHeight int32     `json:"nLastTakeoverheight"`
	Claims             []Claim   `json:"claims"`
	UnmatchedSupports  []Support `json:"unmatched supports"`
}

// Claim models the data of a claim both static and dynamic. Used for claimtrie sync.
type Claim struct {
	Name            string    `json:"name,omitempty"`
	ClaimID         string    `json:"claimId"`
	TxID            string    `json:"txid"`
	N               int32     `json:"n"`
	Height          int32     `json:"nHeight"`
	ValidAtHeight   int32     `json:"nValidAtHeight"`
	Amount          float64   `json:"nAmount"`
	EffectiveAmount float64   `json:"nEffectiveAmount"`
	Supports        []Support `json:"supports,omitempty"`
}

// Support models the support information for a claim in the claimtrie of lbrycrd.
type Support struct {
	TxID          string  `json:"txid"`
	N             int32   `json:"n"`
	Height        int32   `json:"nHeight"`
	ValidAtHeight int32   `json:"nValidAtHeight"`
	Amount        float64 `json:"nAmount"`
}
