package twilio

import (
	"github.com/sfreiberg/gotwilio"
	"github.com/sirupsen/logrus"
)

var twilioClient *gotwilio.Twilio

//RecipientList is the list of phone numbers that twilio sends messages to.
var RecipientList = []string{}

//FromNumber is the phone number that text messages come from.
var FromNumber = ""

//TwilioAuthToken is the auth token for twilio account integration.
var TwilioAuthToken = ""

//TwilioSID is the twilio account SID.
var TwilioSID = ""

//InitTwilio initializes the twilio client to send SMS messages from chainquery to a list of numbers.
func InitTwilio() {
	if TwilioAuthToken != "" && TwilioSID != "" {
		twilioClient = gotwilio.NewTwilioClient(TwilioSID, TwilioAuthToken)
	}
}

//SendSMS sends a text message to the recipient list if twilio integration is setup, based on configuration.
func SendSMS(message string) {
	if twilioClient != nil {
		for _, recipient := range RecipientList {
			_, exception, err := twilioClient.SendSMS(FromNumber, recipient, "Chainquery: "+message, "", TwilioSID)
			if err != nil {
				logrus.Error(err)
			}
			if exception != nil {
				logrus.Warning("Status: ", exception.Status, "Code: ", exception.Code, "Message: ", exception.Message, "From Twilio: ", exception.MoreInfo)
			}
		}
	}
}
