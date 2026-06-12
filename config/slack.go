package config

import (
	"fmt"
	"sort"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/spf13/viper"
)

type slackMessagePoster interface {
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)
}

type slackLogHook struct {
	poster  slackMessagePoster
	channel string
	levels  []logrus.Level
}

func InitSlack() {
	slackBotToken := viper.GetString(slackbottoken)
	slackChannel := viper.GetString(slackchannel)
	slackLogLevel := viper.GetInt(slackloglevel)
	if slackBotToken != "" && slackChannel != "" {
		logrus.AddHook(newSlackLogHook(slack.New(slackBotToken), slackChannel, logrus.Level(slackLogLevel)))
	}
}

func newSlackLogHook(poster slackMessagePoster, channel string, level logrus.Level) *slackLogHook {
	return &slackLogHook{
		poster:  poster,
		channel: channel,
		levels:  slackAcceptedLevels(level),
	}
}

func (hook *slackLogHook) Levels() []logrus.Level {
	return hook.levels
}

func (hook *slackLogHook) Fire(entry *logrus.Entry) error {
	options := []slack.MsgOption{
		slack.MsgOptionText(entry.Message, false),
	}
	attachments := slackLogAttachments(entry)
	if len(attachments) > 0 {
		options = append(options, slack.MsgOptionAttachments(attachments...))
	}
	_, _, err := hook.poster.PostMessage(
		hook.channel,
		options...,
	)
	return err
}

func slackAcceptedLevels(level logrus.Level) []logrus.Level {
	levels := make([]logrus.Level, 0, len(logrus.AllLevels))
	for _, candidate := range logrus.AllLevels {
		if candidate <= level {
			levels = append(levels, candidate)
		}
	}
	return levels
}

func slackLogAttachments(entry *logrus.Entry) []slack.Attachment {
	fields := slackLogFields(entry.Data)
	if len(fields) == 0 {
		return nil
	}
	return []slack.Attachment{{
		Color:    slackLogColor(entry.Level),
		Fallback: entry.Message,
		Text:     "Message fields",
		Fields:   fields,
	}}
}

func slackLogColor(level logrus.Level) string {
	switch level {
	case logrus.DebugLevel:
		return "#9B30FF"
	case logrus.InfoLevel:
		return "good"
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return "danger"
	default:
		return "warning"
	}
}

func slackLogFields(data logrus.Fields) []slack.AttachmentField {
	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := make([]slack.AttachmentField, 0, len(keys))
	for _, key := range keys {
		value := fmt.Sprint(data[key])
		fields = append(fields, slack.AttachmentField{
			Title: key,
			Value: value,
			Short: len(value) <= 20,
		})
	}
	return fields
}
