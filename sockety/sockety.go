package sockety

import (
	"github.com/lbryio/chainquery/metrics"
	"github.com/lbryio/errors.go"
	"github.com/lbryio/sockety/socketyapi"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null"
)

// Token token used to sent notifications to sockety
var Token string

// URL is the url to connect to an instance of sockety.
var URL string

var socketyClient *socketyapi.Client

// SendNotification sends the notification to socket using client
func SendNotification(args socketyapi.SendNotificationArgs) {
	if Token == "" || URL == "" {
		return
	}
	defer catchPanic()
	if socketyClient == nil {
		logrus.Debug("initializating sockety client")
		socketyClient = socketyapi.NewClient(URL, Token)
	}
	_, err := socketyClient.SendNotification(args)
	if err != nil {
		logrus.Error(errors.Prefix("Socket Send Notification:", err))
	}
	metrics.SocketyNotifications.WithLabelValues(args.Type, null.StringFromPtr(args.Category).String, null.StringFromPtr(args.SubCategory).String).Inc()
}

func catchPanic() {
	if r := recover(); r != nil {
		logrus.Error("sockety send recovered from: ", r)
	}
}
