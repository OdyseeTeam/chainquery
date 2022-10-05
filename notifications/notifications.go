package notifications

import (
	"io"
	"net/http"
	"net/url"

	"github.com/lbryio/chainquery/metrics"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/sirupsen/logrus"
)

type subscriber struct {
	URL    string
	Type   string
	Params url.Values
}

var subscriptions map[string][]subscriber

// AddSubscriber adds a subscriber to the subscribers list for a type
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

// ClearSubscribers clears the list of subscribers
func ClearSubscribers() {
	subscriptions = make(map[string][]subscriber)
}

// Notify notifies the list of subscribers for a type
func Notify(t string, values url.Values) {
	subs, ok := subscriptions[t]
	if ok {
		for _, s := range subs {
			for param, value := range s.Params {
				values.Set(param, value[0])
			}
			s.notify(values)
			metrics.Notifications.WithLabelValues(t).Inc()
		}
	}
}

func (s subscriber) notify(values url.Values) {
	res, err := http.PostForm(s.URL, values)
	if err != nil {
		logrus.Error(errors.Prefix("Notify", errors.Err(err)))
	}
	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Error(errors.Prefix("Notify", errors.Err(err)))
		}
	}()
	b, err := io.ReadAll(res.Body)
	// b, err := ioutil.ReadAll(resp.Body)  Go.1.15 and earlier
	if err != nil {
		logrus.Error(errors.Prefix("Notify", errors.Err(err)))
	}

	logrus.Errorln(string(b))
}
