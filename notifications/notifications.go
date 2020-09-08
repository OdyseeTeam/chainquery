package notifications

import (
	"net/http"
	"net/url"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/sirupsen/logrus"
)

type subscriber struct {
	URL    string
	Type   string
	Params url.Values
}

var subscriptions map[string][]subscriber

func AddSubscriber(address, subType string, params map[string]interface{}) {
	if subscriptions == nil {
		subscriptions = make(map[string][]subscriber)
	}
	urlParams := url.Values{}
	for param, v := range params {
		value, ok := v.(string)
		if ok {
			urlParams.Set(param, value)
		}
	}
	subscriptions[subType] = append(subscriptions[subType], subscriber{URL: address, Type: subType, Params: urlParams})
}

func ClearSubscribers() {
	subscriptions = make(map[string][]subscriber)
}

func Notify(t string, values url.Values) {
	subs, ok := subscriptions[t]
	if ok {
		for _, s := range subs {
			for param, value := range s.Params {
				values.Set(param, value[0])
			}
			s.notify(values)
		}
	}
}

func (s subscriber) notify(values url.Values) {
	_, err := http.PostForm(s.URL, values)
	if err != nil {
		logrus.Error(errors.Prefix("Notify:", errors.Err(err)))
	}
}
