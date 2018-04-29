package lbrycrd

import (
	"encoding/json"

	"github.com/lbryio/lbry.go/errors"
)

// V1Claim is the first version of claim metadata used by lbry.
type V1Claim struct {
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

// V2Claim is the second version of claim metadata used by lbry.
type V2Claim struct {
	Version     string  `json:"ver"`          //Required
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
	LicenseURL  *string `json:"license_url,omitempty"`
	NSFW        bool    `json:"nsfw"` //Required

}

// V3Claim is the third version of claim metadata used by lbry.
type V3Claim struct {
	Version     string  `json:"ver"`          //Required
	Title       string  `json:"title"`        //Required
	Description string  `json:"description"`  //Required
	Author      string  `json:"author"`       //Required
	Language    string  `json:"language"`     //Required
	License     string  `json:"license"`      //Required
	Sources     Sources `json:"sources"`      //Required
	ContentType string  `json:"content_type"` //Required
	Thumbnail   *string `json:"thumbnail,omitempty"`
	Fee         *Fee    `json:"fee,omitempty"`
	Contact     *int    `json:"contact,omitempty"`
	PubKey      *string `json:"pubkey,omitempty"`
	LicenseURL  *string `json:"license_url,omitempty"`
	NSFW        bool    `json:"nsfw"` //Required
	Sig         *string `json:"sig"`
}

// FeeInfo is the structure of fee information used by lbry.
type FeeInfo struct {
	Amount  float32 `json:"amount"`  //Required
	Address string  `json:"address"` //Required
}

// Sources is the structure of Sources that can be used for a claim. Sources mainly include lbrysdhash but could be from
// elsewhere in the future.
type Sources struct {
	LbrySDHash string `json:"lbry_sd_hash"` //Required
	BTIH       string `json:"btih"`         //Required
	URL        string `json:"url"`          //Required
}

// Fee is the structure used for different currencies allowed for claims.
type Fee struct {
	LBC *FeeInfo `json:"LBC,omitempty"`
	BTC *FeeInfo `json:"BTC,omitempty"`
	USD *FeeInfo `json:"USD,omitempty"`
}

// Unmarshal is an implementation to unmarshal the V1 claim from json. Main addition is to check the version.
func (c *V1Claim) Unmarshal(value []byte) error {
	err := json.Unmarshal(value, c)
	if err != nil {
		return err
	} //Version can be blank for version 1
	if c.Version != "" && c.Version != "0.0.1" {
		err = errors.Base("Incorrect version, expected 0.0.1 found " + c.Version)
		return err
	}
	//ToDo - restrict to required fields?

	return nil
}

// Unmarshal is an implementation to unmarshal the V2 claim from json. Main addition is to check the version.
func (c *V2Claim) Unmarshal(value []byte) error {
	err := json.Unmarshal(value, c)
	if err != nil {
		return err
	}
	if c.Version != "0.0.2" {
		err = errors.Base("Incorrect version, expected 0.0.2 found " + c.Version)
		return err
	}

	return nil
}

// Unmarshal is an implementation to unmarshal the V3 claim from json. Main addition is to check the version.
func (c *V3Claim) Unmarshal(value []byte) error {
	err := json.Unmarshal(value, c)
	if err != nil {
		return err
	}
	if c.Version != "0.0.3" {
		err = errors.Base("Incorrect version, expected 0.0.3 found " + c.Version)
		return err
	}

	return nil
}
