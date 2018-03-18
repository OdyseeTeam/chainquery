package processing

type TxDebitCredits struct {
	addresses map[string]AddrDebitCredits
}

type AddrDebitCredits struct {
	debits  float64
	credits float64
}
