package lbrycrd

import (
	"encoding/json"

	"github.com/lbryio/lbry.go/errors"
)

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
