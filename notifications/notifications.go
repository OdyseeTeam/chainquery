package notifications

import (
	"net/http"
	"net/url"
	"sync"
	"time"

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
var subscriptionsMu sync.RWMutex
var notificationWorkerOnce sync.Once
var notificationQueue chan notificationJob
var notificationClient *http.Client
var notificationClientTimeout time.Duration
var notificationClientMu sync.Mutex
var Timeout = 20 * time.Second

type notificationJob struct {
	Type   string
	Values url.Values
}

// AddSubscriber adds a subscriber to the subscribers list for a type
func AddSubscriber(address, subType string, params map[string]interface{}) {
	subscriptionsMu.Lock()
	defer subscriptionsMu.Unlock()
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
	subscriptionsMu.Lock()
	defer subscriptionsMu.Unlock()
	subscriptions = make(map[string][]subscriber)
}

// Notify notifies the list of subscribers for a type
func Notify(t string, values url.Values) {
	notificationWorkerOnce.Do(startNotificationWorkers)
	job := notificationJob{Type: t, Values: copyValues(values)}
	select {
	case notificationQueue <- job:
	default:
		logrus.Warn("notification queue full; dropping notification")
	}
}

func startNotificationWorkers() {
	notificationQueue = make(chan notificationJob, 1000)
	for worker := 0; worker < 20; worker++ {
		go notificationWorker()
	}
}

func notificationWorker() {
	for job := range notificationQueue {
		processNotification(job)
	}
}

func processNotification(job notificationJob) {
	subscriptionsMu.RLock()
	subs := append([]subscriber(nil), subscriptions[job.Type]...)
	subscriptionsMu.RUnlock()
	for _, s := range subs {
		values := copyValues(job.Values)
		for param, value := range s.Params {
			values.Set(param, value[0])
		}
		s.notify(values)
		metrics.Notifications.WithLabelValues(job.Type).Inc()
	}
}

func (s subscriber) notify(values url.Values) {
	res, err := notificationHTTPClient().PostForm(s.URL, values)
	if err != nil {
		logrus.Error(errors.Prefix("Notify", errors.Err(err)))
		return
	}
	defer func() {
		err = res.Body.Close()
		if err != nil {
			logrus.Error(errors.Prefix("Notify", errors.Err(err)))
		}
	}()
	//b, err := io.ReadAll(res.Body)
	//if err != nil {
	//	logrus.Error(errors.Prefix("Notify", errors.Err(err)))
	//}
	//if res.StatusCode != http.StatusOK {
	//	logrus.Errorln(string(b))
	//}
}

func notificationHTTPClient() *http.Client {
	notificationClientMu.Lock()
	defer notificationClientMu.Unlock()
	if notificationClient == nil || notificationClientTimeout != Timeout {
		notificationClient = &http.Client{Timeout: Timeout}
		notificationClientTimeout = Timeout
	}
	return notificationClient
}

func copyValues(values url.Values) url.Values {
	copied := make(url.Values, len(values))
	for key, value := range values {
		copied[key] = append([]string(nil), value...)
	}
	return copied
}
