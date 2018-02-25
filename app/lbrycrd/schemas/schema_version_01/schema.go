package schema_version_01

type Claim struct {
	Version     string  `json:"ver,omitempty"`
	Title       string  `json:"title"`        //Required
	Description string  `json:"description"`  //Required
	Author      string  `json:"author"`       //Required
	Language    string  `json:"language"`     //Required
	License     string  `json:"license"`      //Required
	Sources     Sources `json:"sources"`      //Required
	ContentType string  `json:"content-type"` //Required
	Thumbnail   string  `json:"thumbnail,omitempty"`
	Fee         Fee     `json:"fee,omitempty"`
	Contact     int     `json:"contact,omitempty"`
	PubKey      string  `json:"pubkey,omitempty"`
}

type FeeInfo struct {
	Amount  int    `json:amount`  //Required
	Address string `json:address` //Required
}

type Sources struct {
	LbrySDHash string `json:"lbry_sd_hash"` //Required
	BTIH       string `json:"btih"`         //Required
	URL        string `json:"url"`          //Required
}

type Fee struct {
	LBC FeeInfo `json:"fee_info,omitempty"`
	BTC FeeInfo `json:"fee_info,omitempty"`
	USD FeeInfo `json:"fee_info,omitempty"`
}
