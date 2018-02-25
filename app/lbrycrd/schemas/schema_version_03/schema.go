package schema_version_03

import (
	v1 "github.com/lbryio/chainquery/app/lbrycrd/schemas/schema_version_01"
)

type Claim struct {
	Version     string     `json:"ver"`          //Required
	Title       string     `json:"title"`        //Required
	Description string     `json:"description"`  //Required
	Author      string     `json:"author"`       //Required
	Language    string     `json:"language"`     //Required
	License     string     `json:"license"`      //Required
	Sources     v1.Sources `json:"sources"`      //Required
	ContentType string     `json:"content-type"` //Required
	Thumbnail   *string    `json:"thumbnail,omitempty"`
	Fee         *v1.Fee    `json:"fee,omitempty"`
	Contact     *int       `json:"contact,omitempty"`
	PubKey      *string    `json:"pubkey,omitempty"`
	LicenseURL  *string    `json:"license_url,omitempty"`
	NSFW        bool       `json:"nsfw"` //Required
	Sig         *string    `json:"sig"`
}
