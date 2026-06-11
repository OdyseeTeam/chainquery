package sockety

import (
	"sync"
	"time"

	"github.com/lbryio/chainquery/metrics"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"github.com/OdyseeTeam/sockety/socketyapi"
	"github.com/sirupsen/logrus"
	"github.com/volatiletech/null/v8"
)

// Token token used to sent notifications to sockety
var Token string

// URL is the url to connect to an instance of sockety.
var URL string

var socketyClient *socketyapi.Client
var socketyClientURL string
var socketyClientToken string
var socketyClientMu sync.Mutex
var socketyWorkerOnce sync.Once
var socketyQueue chan socketyapi.SendNotificationArgs
var Timeout = 20 * time.Second

// SendNotification sends the notification to socket using client
func SendNotification(args socketyapi.SendNotificationArgs) {
	if Token == "" || URL == "" {
		return
	}
	socketyWorkerOnce.Do(startSocketyWorkers)
	select {
	case socketyQueue <- args:
	default:
		logrus.Warn("sockety notification queue full; dropping notification")
	}
}

func startSocketyWorkers() {
	socketyQueue = make(chan socketyapi.SendNotificationArgs, 1000)
	for worker := 0; worker < 4; worker++ {
		go socketyWorker()
	}
}

func socketyWorker() {
	for args := range socketyQueue {
		sendNotification(args)
	}
}

func sendNotification(args socketyapi.SendNotificationArgs) {
	defer catchPanic()
	client := getSocketyClient()
	_, err := client.SendNotification(args)
	if err != nil {
		logrus.Error(errors.Prefix("Socket Send Notification", err))
	}
	metrics.SocketyNotifications.WithLabelValues(args.Type, null.StringFromPtr(args.Category).String, null.StringFromPtr(args.SubCategory).String).Inc()
}

func getSocketyClient() *socketyapi.Client {
	socketyClientMu.Lock()
	defer socketyClientMu.Unlock()
	if socketyClient == nil || socketyClientURL != URL || socketyClientToken != Token {
		logrus.Debug("initializing sockety client")
		socketyClient = socketyapi.NewClient(URL, Token)
		socketyClient.Timeout = Timeout
		socketyClientURL = URL
		socketyClientToken = Token
	}
	return socketyClient
}

func catchPanic() {
	if r := recover(); r != nil {
		logrus.Error("sockety send recovered from: ", r)
	}
}
