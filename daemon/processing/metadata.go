package processing

import (
	json2 "encoding/json"

	"github.com/lbryio/lbryschema.go/claim"
)

// value.stream.metadata.author
// value.stream.metadata.title
// value.stream.metadata.description
// value.claimType
// value.stream.source.contentType
// value.stream.metadata.nsfw

//Value holds the current structure for metadata as json that is ensures a major versions backwards compatibility.
type Value struct {
	Claim *Claim `json:"Claim"`
}

//Claim holds the current Claim structure
type Claim struct {
	ClaimType string  `json:"claimType"`
	Stream    *Stream `json:"stream"`
}

//Stream holds the metadata and source
type Stream struct {
	Metadata *Metadata `json:"metadata"`
	Source   *Source   `json:"source"`
}

//Metadata holds information that is used via the table column it is assigned for backwards compatibility.
type Metadata struct {
	Author      string `json:"author"`
	Title       string `json:"title"`
	Description string `json:"description"`
	NSFW        bool   `json:"nsfw"`
}

//Source holds the media type of the claim
type Source struct {
	ContentType string `json:"contentType"`
}

const streamType = "streamType"
const certificateType = "certificateType"

//GetValueAsJSON returns the JSON string of the structure of claim metadata.
func GetValueAsJSON(helper claim.ClaimHelper) (string, error) {
	var value Value
	if helper.GetStream() != nil {
		s := helper.GetStream()
		contentType := ""
		if s.GetSource() != nil {
			contentType = s.GetSource().GetMediaType()
		}
		nsfw := tagExists("mature", helper.GetTags())
		value = Value{
			&Claim{
				streamType,
				&Stream{
					&Metadata{
						s.GetAuthor(),
						helper.GetTitle(),
						helper.GetDescription(),
						nsfw,
					},
					&Source{
						contentType,
					},
				},
			},
		}
	} else if helper.GetChannel() != nil {
		value = Value{&Claim{certificateType, nil}}
	}

	json, err := json2.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(json), nil
}

func tagExists(tagName string, taglist []string) bool {
	for _, tag := range taglist {
		if tag == tagName {
			return true
		}
	}
	return false
}
