package schema_version_03

import (
	"encoding/json"

	v1 "github.com/lbryio/chainquery/lbrycrd/schemas/schema_version_01"
	"github.com/lbryio/lbry.go/errors"
)

var version = "0.0.2"

type Claim struct {
	Version     string     `json:"ver"`          //Required
	Title       string     `json:"title"`        //Required
	Description string     `json:"description"`  //Required
	Author      string     `json:"author"`       //Required
	Language    string     `json:"language"`     //Required
	License     string     `json:"license"`      //Required
	Sources     v1.Sources `json:"sources"`      //Required
	ContentType string     `json:"content_type"` //Required
	Thumbnail   *string    `json:"thumbnail,omitempty"`
	Fee         *v1.Fee    `json:"fee,omitempty"`
	Contact     *int       `json:"contact,omitempty"`
	PubKey      *string    `json:"pubkey,omitempty"`
	LicenseURL  *string    `json:"license_url,omitempty"`
	NSFW        bool       `json:"nsfw"` //Required
	Sig         *string    `json:"sig"`
}

func (c *Claim) Unmarshal(value []byte) error {
	err := json.Unmarshal(value, c)
	if err != nil {
		return err
	}
	if c.Version != version {
		err = errors.Base("Incorrect version, expected " + version + " found " + c.Version)
		return err
	}

	return nil
}
