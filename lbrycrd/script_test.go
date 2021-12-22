package lbrycrd

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/lbryio/chainquery/util"

	"github.com/btcsuite/btcd/txscript"

	"github.com/lbryio/chainquery/global"
)

type HashAddressPair struct {
	hash    string
	address string
}

var P2PKHPairs = []HashAddressPair{
	{"5f7a5a5aab24884b74639e221388e443f1a0a5ef", "bMS7TgmB7CUNB7FsimV2wi27YUNSpTNdSo"},
	{"36c63c9af872095dc7bc5a6adab80b54f7a12e3c", "bHitfKVqDQH8hwsFcSWKNGa7MMoMJf4Cm3"},
	{"e7928d0fdff4f46473e1c83b2721873051d403e9", "bZqiM61b6NBQ3uv6gfKLPHWiNiDyd128Jd"},
	{"244b19c5f733ccf596876f9328c016f24bcd9478", "bG3AzDtxRxFZ3Nv1CYHmvRiZ7voQndauhy"},
	{"1be1ec470deb59fc69850cf05e787f87175a8244", "bFGhap5jZYbnz5UQUHrkGmTh4NiRvjUE4f"},
}
var P2PKPairs = []HashAddressPair{
	{"024ca653fc094c95aa409430caf2eee08fa6e5fbbe78431e0ec9e7cd80193d98f9", "bZi1WEjGtsdAwuZTnNNTCAZLxhHkiHec4m"},
	{"044ca653fc094c95aa409430caf2eee08fa6e5fbbe78431e0ec9e7cd80193d98f991b8e88792b46d622d128b146e7aca49fbbf858f1e7e452b0e7ae556d5b4556e", "bRpUYMFSHGASCEAW22cVCf4iFeKB2BHEq9"}}
var P2SHPairs = []HashAddressPair{
	{"a6e68448580140c4861a920c7d5140065d45e14b", "rMT5Sg8SyFP3ax2PRaweRCRZoMeYw4znEi"},
	{"6c4aab30dc6cd9c07c40a598f2ee5f41bea3b750", "rG7BZ3EmPMLcggYYkRTveXv8pqedWPDG7p"},
	{"599885176d5d868c72f7327f573f37b4f91d0fa6", "rEQKyb7nd7UUGyEEn5xRkk1fgXdTCf2ZCg"},
	{"20b7bd1bc21a55cbf6b2d554eb48b669eb6d1263", "r9DarmxyPjWkF7ocyxMzaNZN3a9gJvNTZJ"},
}

var P2WPKHPairs = []HashAddressPair{ //From Testnet
	{"1892d4c5b69ba764bcf68bc43a9359472c4e18a0", "tlbc1qrzfdf3dknwnkf08k30zr4y6egukyux9qe04vch"},
}

func TestAddressExtraction(t *testing.T) {
	//Should add main net examples when live.
	global.BlockChainName = lbrycrdTestnet
	chainParams, err := GetChainParams()
	if err != nil {
		t.Error(err)
	}
	scriptHex := "00141892d4c5b69ba764bcf68bc43a9359472c4e18a0"
	script, err := hex.DecodeString(scriptHex)
	if err != nil {
		t.Error(err)
	}
	class, address, reSigs, err := txscript.ExtractPkScriptAddrs(script, chainParams)
	if reSigs != 1 {
		t.Errorf("Expected 1 sig required but returned %d", reSigs)
	}
	if len(address) != 1 {
		t.Error("expected on 1 address returned")
	}
	if address[0].EncodeAddress() != "tlbc1qrzfdf3dknwnkf08k30zr4y6egukyux9qe04vch" {
		t.Errorf("expected address 'tlbc1qrzfdf3dknwnkf08k30zr4y6egukyux9qe04vch' but got '%s'", address)
	}
	println("Class:", class)
}

func TestGetAddressFromP2WPKH(t *testing.T) {
	//Should add main net examples when live.
	global.BlockChainName = lbrycrdTestnet
	for _, pair := range P2WPKHPairs {
		result, err := getAddressFromP2WPKH(pair.hash)
		if err != nil {
			t.Error(err)
		}
		if result != pair.address {
			t.Errorf("expected '%s' but got '%s' instead", pair.address, result)
		}
	}
	global.BlockChainName = lbrycrdMain
}

func TestGetAddressFromP2PKH(t *testing.T) {
	for _, pair := range P2PKHPairs {
		good := pair.address
		result, err := getAddressFromP2PKH(pair.hash)
		if err != nil {
			t.Error(err)
		} else if result != good {
			t.Error("Failure - Expected ", good, " got ", result)
		}
	}
}

func TestGetAddressFromP2PK(t *testing.T) {
	for _, pair := range P2PKPairs {
		good := pair.address
		result, err := getAddressFromP2PK(pair.hash)
		if err != nil {
			t.Error(err)
		} else if result != good {
			t.Error("Failure - Expected ", good, " got ", result)
		}
	}
}

func TestGetAddressFromP2SH(t *testing.T) {
	for _, pair := range P2SHPairs {
		good := pair.address
		result, err := getAddressFromP2SH(pair.hash)
		if err != nil {
			t.Error(err)
		} else if result != good {
			t.Error("Failure - Expected ", good, " got ", result)
		}
	}
}

// Testing that a larger than 75 byte name parses corrected.
func TestParseClaimNameScript1(t *testing.T) {
	scriptHex := "b54c547365677769742d636f64652d69732d72656c65617365642d616e642d7468652d706972617465732d6f662d6963656c616e642d616476616e63652d7468652d63727970746f76657273652d3133332d766964656f4de8057b22666565223a207b22555344223a207b22616d6f756e74223a20302e312c202261646472657373223a2022624e4e79624362333731533768756a716a75314c354738686270613862454542716d227d7d2c2022766572223a2022302e302e33222c20226465736372697074696f6e223a20224f6e20746f646179277320657069736f6465206f66205468652043727970746f76657273653a5c6e4963656c616e642773205069726174652050617274792068617320747269706c65642069747320736561747320696e207468652036332d73656174207061726c69616d656e742c20656c656374696f6e20726573756c74732073686f7720616e642074686520426974636f696e20436f7265207465616d206861732072656c6561736564206974732076657273696f6e20302e31332e31207570646174652e5c6e5c6e546f646179277320657069736f64652069732073706f6e736f72656420627920446173682c20746865207072697661637920666f6375736564206469676974616c2063757272656e63792074686174206f6666657273207472616e73616374696f6e73207769746820696e7374616e7420636f6e6669726d6174696f6e732e205b436c69636b206865726520746f206c6561726e206d6f72655d28687474703a2f2f6269742e6c792f32663939473762295c6e5c6e536f75726365733a5c6e5b546865204242432041727469636c652041626f757420546865205069726174652050617274795d28687474703a2f2f7777772e6262632e636f2e756b2f6e6577732f776f726c642d6575726f70652d3337383133353634295c6e5c6e5b546865204f726967696e616c2041727469636c65204f6e20426974636f696e2e636f6d5d2868747470733a2f2f6e6577732e626974636f696e2e636f6d2f626974636f696e2d636f72652d302d31332d312d72656c65617365642d7365677769742f295c6e5c6e5b4c69766520537461747573206f6620556e636f6e6669726d6174696f6e205472616e73616374696f6e73206f6e2074686520426974636f696e204e6574776f726b5d2868747470733a2f2f626c6f636b636861696e2e696e666f2f756e636f6e6669726d65642d7472616e73616374696f6e73295c6e5c6e50726f64756365642062792043727970746f766572736974792e636f6d20746865206f6e6c696e65207363686f6f6c20666f72206c6561726e696e672061626f757420426974636f696e2c2063727970746f2d63757272656e6369657320616e6420626c6f636b636861696e732e5c6e5c6e68747470733a2f2f7777772e63727970746f766572736974792e636f6d2f222c20226c6963656e7365223a2022437265617469766520436f6d6d6f6e73204174747269627574696f6e20342e3020496e7465726e6174696f6e616c222c2022617574686f72223a2022436872697320436f6e6579222c20227469746c65223a202253656757697420436f64652049732052656c656173656420616e64205468652050697261746573206f66204963656c616e6420416476616e636520285468652043727970746f7665727365202331333329222c20226c616e6775616765223a2022656e222c20226c6963656e73655f75726c223a202268747470733a2f2f6372656174697665636f6d6d6f6e732e6f72672f6c6963656e7365732f62792f342e302f6c6567616c636f6465222c2022636f6e74656e745f74797065223a2022766964656f2f6d7034222c20226e736677223a2066616c73652c2022736f7572636573223a207b226c6272795f73645f68617368223a2022333536623366383933366262396636333932396162396266383064653530666230623437316239393432396164633265323962333632666566626136646661376339616431383866383961363936346237343433646665666131623931316262227d2c20227468756d626e61696c223a202268747470733a2f2f737465656d69742d6275636b65742d34613734336563322e73332e616d617a6f6e6177732e636f6d2f33312d31302d323031362532302d2532307468756d622d6f70742e706e67227d6d7576a914028d35d0b2a1833208a87cebe5e592c37ffb37ac88ac"
	correctName := "segwit-code-is-released-and-the-pirates-of-iceland-advance-the-cryptoverse-133-video"
	scriptBytes, err := hex.DecodeString(scriptHex)
	if err != nil {
		t.Error(err)
	}
	name, _, _, err := ParseClaimNameScript(scriptBytes)
	if err != nil {
		t.Error(err)
	}
	if name != correctName {
		t.Error("Parse error for claim name: expected ", correctName, " got ", name)
	}
}

// Testing that a less than 76 byte name parses corrected.
func TestParseClaimNameScript2(t *testing.T) {
	scriptHex := "b506676f6f676c654dd0017b22766572223a2022302e302e33222c20226465736372697074696f6e223a20226d6520616e6420406b6c69707a6f6f2061726520747279696e6720746f20666967757265206f757420686f77207468697320776f726b73222c20226c6963656e7365223a2022437265617469766520436f6d6d6f6e73204174747269627574696f6e20342e3020496e7465726e6174696f6e616c222c2022617574686f72223a20224e696b6f6f6f373737222c20227469746c65223a20225468697320697320612074657374222c20226c616e6775616765223a2022656e222c20226e736677223a2066616c73652c2022636f6e74656e745f74797065223a2022696d6167652f706e67222c20226c6963656e73655f75726c223a202268747470733a2f2f6372656174697665636f6d6d6f6e732e6f72672f6c6963656e7365732f62792f342e302f6c6567616c636f6465222c2022736f7572636573223a207b226c6272795f73645f68617368223a2022313564623662343761666363646536363933396131353639303765656638616134316239666233353664643439396138343964663566656464313837636264333734326132623232653539663438356263346561626364636666383739663762227d7d6d7576a914c42a72a4a553138b1ae1270de25283b37966e54888ac"
	correctName := "google"
	scriptBytes, err := hex.DecodeString(scriptHex)
	if err != nil {
		t.Error(err)
	}
	name, _, _, err := ParseClaimNameScript(scriptBytes)
	if err != nil {
		t.Error(err)
	}
	if name != correctName {
		t.Error("Parse error for claim name: expected ", correctName, " got ", name)
	}
}

func TestPurchaseScriptParse(t *testing.T) {
	hexStr := "6a17500a14b5fb292f0ccb678a0c393b5ab47c522d1a9f4bfc"
	hexBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatal(err)
	}
	isPurchase := IsPurchaseScript(hexBytes)
	if !isPurchase {
		t.Fatal("test string no longer identifies as a purchase!")
	}
	purchase, err := ParsePurchaseScript(hexBytes)
	if err != nil {
		t.Fatal(err)
	}
	purchase.GetClaimHash()
	bytes := util.ReverseBytes(purchase.GetClaimHash())
	claimID := hex.EncodeToString(bytes)
	expectedClaimID := "fc4b9f1a2d527cb45a3b390c8a67cb0c2f29fbb5"
	if claimID != expectedClaimID {
		t.Errorf("expected %s, got %s", expectedClaimID, claimID)
	}
}

func TestParseClaimSupportScript(t *testing.T) {
	scriptHex := "b609405363694669344d6514cf5290ec6c4eebbd5c2fcf833944335526a0a63f4c550189e52371184a4100f40f786a151d13be53a97ecb168c690ebb97199b3a1a4aae3ba078f1c2b3bd8cc057e6a8d91078f8a09a6f945dabacd12faed9f4b2649e1ae55eb02a294d907447a6f5e12ac76a1c5ad278a46d6d76a9140e570ed0ef92cd798e6dabc05d6b4923eb0049a988ac"
	scriptBytes, err := hex.DecodeString(scriptHex)
	if err != nil {
		t.Fatal(err)
	}
	name, claimid, value, pkscript, err := ParseClaimSupportScript(scriptBytes)
	if err != nil {
		t.Fatal(err)
	}
	println(fmt.Sprintf("Name: %s\n ClaimID: %s\n ValueHex: %x\n PKSCriptHex: %x", name, claimid, value, pkscript))
}
