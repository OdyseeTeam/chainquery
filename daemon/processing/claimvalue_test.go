package processing

import (
	"encoding/hex"
	"testing"
)

type valueTestPair struct {
	ValueAsHex string
	Claim      claimResult
}

type claimResult struct {
	Author      string
	Title       string
	Description string
	License     string
	FeeAmount   float32
	FeeCurrency string
	FeeAddress  string
	ContentType string
	Language    string
	LbrySDHash  string
	Thumbnail   string
}

var jsonVersion1Tests = []valueTestPair{
	{"7b22666565223a207b224c4243223a207b22616d6f756e74223a20312e302c202261646472657373223a2022625077474139683775696a6f79357541767a565051773951794c6f595a6568484a6f227d7d2c20226465736372697074696f6e223a2022313030304d4220746573742066696c6520746f206d65617375726520646f776e6c6f6164207370656564206f6e204c627279207032702d6e6574776f726b2e222c20226c6963656e7365223a20224e6f6e65222c2022617574686f72223a2022726f6f74222c20226c616e6775616765223a2022456e676c697368222c20227469746c65223a2022313030304d4220737065656420746573742066696c65222c2022736f7572636573223a207b226c6272795f73645f68617368223a2022626439343033336431336634663339303837303837303163616635363562666130396366616466326633346661646634613733666238366232393564316232316137653634383035393934653435623566626336353066333062616334383734227d2c2022636f6e74656e742d74797065223a20226170706c69636174696f6e2f6f637465742d73747265616d222c20227468756d626e61696c223a20222f686f6d65726f626572742f6c6272792f73706565642e6a7067227d",
		claimResult{"root",
			"1000MB speed test file",
			"1000MB test file to measure download speed on Lbry p2p-network.",
			"None",
			1,
			"LBC",
			"bPwGA9h7uijoy5uAvzVPQw9QyLoYZehHJo",
			"'application/octet-stream",
			"en",
			"bd94033d13f4f3908708701caf565bfa09cfadf2f34fadf4a73fb86b295d1b21a7e64805994e45b5fbc650f30bac4874",
			"/homerobert/lbry/speed.jpg"},
	},
}

func TestMigrationFromJSONVersion1(t *testing.T) {
	for _, pair := range jsonVersion1Tests {
		valueBytes, err := hex.DecodeString(pair.ValueAsHex)
		if err != nil {
			t.Error(err)
		}
		claim, err := decodeClaimValue("", valueBytes)
		if err != nil {
			t.Error("Decode error: ", err)
		}
		if claim.GetStream().GetMetadata().GetAuthor() != pair.Claim.Author {
			t.Error("Author mismatch: expected", pair.Claim.Author, "got", claim.GetStream().GetMetadata().GetAuthor())
		}
		if claim.GetStream().GetMetadata().GetTitle() != pair.Claim.Title {
			t.Error("Title mismatch: expected", pair.Claim.Title, "got", claim.GetStream().GetMetadata().GetTitle())
		}
		if claim.GetStream().GetMetadata().GetDescription() != pair.Claim.Description {
			t.Error("Description mismatch: expected", pair.Claim.Description, "got", claim.GetStream().GetMetadata().GetDescription())
		}
		if claim.GetStream().GetMetadata().GetLicense() != pair.Claim.License {
			t.Error("License mismatch: expected", pair.Claim.License, "got", claim.GetStream().GetMetadata().GetLicense())
		}
		if claim.GetStream().GetMetadata().GetFee().GetAmount() != pair.Claim.FeeAmount {
			t.Error("Fee Amount mismatch: expected", pair.Claim.FeeAmount, "got", claim.GetStream().GetMetadata().GetFee().GetAmount())
		}
		if claim.GetStream().GetMetadata().GetFee().GetCurrency().String() != pair.Claim.FeeCurrency {
			t.Error("Fee Currency mismatch: expected", pair.Claim.FeeCurrency, "got", claim.GetStream().GetMetadata().GetFee().GetCurrency())
		}
		hexaddress := string(claim.GetStream().GetMetadata().GetFee().GetAddress())
		if hexaddress != pair.Claim.FeeAddress {
			t.Error("Fee Address mismatch: expected", pair.Claim.FeeAddress, "got", hexaddress)
		}
		if claim.GetStream().GetSource().GetContentType() == pair.Claim.ContentType {
			t.Error("ContentType mismatch: expected", pair.Claim.ContentType, "got", claim.GetStream().GetSource().GetContentType())
		}
		if claim.GetStream().GetMetadata().GetLanguage().String() == pair.Claim.Language {
			t.Error("Language mismatch: expected", pair.Claim.Language, "got", claim.GetStream().GetMetadata().GetLanguage())
		}
		content := string(claim.GetStream().GetSource().GetSource())
		if content == pair.Claim.LbrySDHash {
			t.Error("Source mismatch: expected", pair.Claim.LbrySDHash, "got", content)
		}
	}
}
