package processing

type txDebitCredits struct {
	AddrDCMap map[string]*AddrDebitCredits
}

func NewTxDebitCredits() txDebitCredits {
	t := txDebitCredits{}
	v := make(map[string]*AddrDebitCredits)
	t.AddrDCMap = v

	return t

}

type AddrDebitCredits struct {
	debits  float64
	credits float64
}

func (addDC AddrDebitCredits) Debits() float64 {
	return addDC.debits
}

func (addDC AddrDebitCredits) Credits() float64 {
	return addDC.credits
}

func (txDC txDebitCredits) subtract(address string, value float64) error {
	if txDC.AddrDCMap[address] == nil {
		addrDC := AddrDebitCredits{}
		txDC.AddrDCMap[address] = &addrDC
	}
	txDC.AddrDCMap[address].debits = txDC.AddrDCMap[address].debits + value
	return nil
}

func (t txDebitCredits) add(address string, value float64) error {
	if t.AddrDCMap[address] == nil {
		addrDC := AddrDebitCredits{}
		t.AddrDCMap[address] = &addrDC
	}
	t.AddrDCMap[address].credits = t.AddrDCMap[address].credits + value

	return nil
}
