package notifications

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func TestAddSubscriber(t *testing.T) {
	viper.SetConfigType("toml")
	err := viper.ReadConfig(strings.NewReader(`
[[subscription.payment]]
url= "http://localhost:8080/event/payment"
auth_token="mytoken"
[[subscription.payment]]
url= "http://localhost:8080/event/payment"
auth_token="mytoken"
`))
	if err != nil {
		t.Error(err)
	}
	subs := viper.GetStringMap("subscription")
	applySubscribers(subs)

}
