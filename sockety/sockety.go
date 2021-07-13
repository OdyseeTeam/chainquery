package sockety

import (
	"github.com/lbryio/errors.go"
	"github.com/lbryio/sockety/socketyapi"
	"github.com/sirupsen/logrus"
)

// SocketyToken token used to sent notifications to sockety
var SocketyToken string

var socketyClient *socketyapi.Client

// SendNotification sends the notification to socket using client
func SendNotification(args socketyapi.SendNotificationArgs) {
	if SocketyToken == "" {
		return
	}
	if socketyClient == nil {
		socketyClient = socketyapi.NewClient("wss://sockety.lbry.com", SocketyToken)
	}
	_, err := socketyClient.SendNotification(args)
	if err != nil {
		logrus.Error(errors.Prefix("Socket Send Notification:", err))
	}
}
