package schema_version_01

import (
	"encoding/json"
	"github.com/lbryio/errors.go"
)

var version = "0.0.1"

type Claim struct {
	Version     string  `json:"ver,omitempty"`
	Title       string  `json:"title"`        //Required
	Description string  `json:"description"`  //Required
	Author      string  `json:"author"`       //Required
	Language    string  `json:"language"`     //Required
	License     string  `json:"license"`      //Required
	Sources     Sources `json:"sources"`      //Required
	ContentType string  `json:"content-type"` //Required
	Thumbnail   *string `json:"thumbnail,omitempty"`
	Fee         *Fee    `json:"fee,omitempty"`
	Contact     *int    `json:"contact,omitempty"`
	PubKey      *string `json:"pubkey,omitempty"`
}

type FeeInfo struct {
	Amount  float32 `json:amount`  //Required
	Address string  `json:address` //Required
}

type Sources struct {
	LbrySDHash string `json:"lbry_sd_hash"` //Required
	BTIH       string `json:"btih"`         //Required
	URL        string `json:"url"`          //Required
}

type Fee struct {
	LBC *FeeInfo `json:"LBC,omitempty"`
	BTC *FeeInfo `json:"BTC,omitempty"`
	USD *FeeInfo `json:"USD,omitempty"`
}

func (c *Claim) Unmarshal(value []byte) error {
	err := json.Unmarshal(value, c)
	if err != nil {
		return err
	} //Version can be blank for version 1
	if c.Version != "" && c.Version != version {
		err = errors.Base("Incorrect version, expected " + version + " found " + c.Version)
		return err
	}
	//ToDo - restrict to required fields?

	return nil
}
