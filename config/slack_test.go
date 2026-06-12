package config

import (
	"errors"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
)

func TestSlackAcceptedLevelsUsesConfiguredThreshold(t *testing.T) {
	levels := slackAcceptedLevels(logrus.WarnLevel)
	expected := []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
	}
	if !reflect.DeepEqual(levels, expected) {
		t.Fatalf("expected levels %v, got %v", expected, levels)
	}
}

func TestSlackLogAttachmentIncludesSortedFields(t *testing.T) {
	entry := &logrus.Entry{
		Data: logrus.Fields{
			"height":               uint64(4),
			"depth":                2,
			"last_matching_height": uint64(1),
		},
		Level:   logrus.WarnLevel,
		Message: "Reorg detected",
	}

	attachments := slackLogAttachments(entry)
	if len(attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(attachments))
	}
	attachment := attachments[0]
	if attachment.Color != "warning" {
		t.Fatalf("expected warning color, got %q", attachment.Color)
	}
	if attachment.Fallback != entry.Message {
		t.Fatalf("expected fallback %q, got %q", entry.Message, attachment.Fallback)
	}
	if attachment.Text != "Message fields" {
		t.Fatalf("expected field header, got %q", attachment.Text)
	}

	expectedFields := []slack.AttachmentField{
		{Title: "depth", Value: "2", Short: true},
		{Title: "height", Value: "4", Short: true},
		{Title: "last_matching_height", Value: "1", Short: true},
	}
	if !reflect.DeepEqual(attachment.Fields, expectedFields) {
		t.Fatalf("expected fields %v, got %v", expectedFields, attachment.Fields)
	}
}

func TestSlackLogAttachmentsWithoutFieldsReturnsNone(t *testing.T) {
	entry := &logrus.Entry{
		Level:   logrus.InfoLevel,
		Message: "Chainquery started",
	}

	attachments := slackLogAttachments(entry)
	if len(attachments) != 0 {
		t.Fatalf("expected no attachments, got %d", len(attachments))
	}
}

func TestSlackLogHookPostsToConfiguredChannel(t *testing.T) {
	poster := &recordingSlackPoster{}
	hook := newSlackLogHook(poster, "C123", logrus.WarnLevel)

	err := hook.Fire(&logrus.Entry{Level: logrus.WarnLevel, Message: "Reorg detected"})
	if err != nil {
		t.Fatal(err)
	}
	if poster.channel != "C123" {
		t.Fatalf("expected channel C123, got %q", poster.channel)
	}
	if poster.calls != 1 {
		t.Fatalf("expected one post, got %d", poster.calls)
	}
}

func TestSlackLogHookReturnsPostError(t *testing.T) {
	expected := errors.New("slack failed")
	poster := &recordingSlackPoster{err: expected}
	hook := newSlackLogHook(poster, "C123", logrus.WarnLevel)

	err := hook.Fire(&logrus.Entry{Level: logrus.WarnLevel, Message: "Reorg detected"})
	if !errors.Is(err, expected) {
		t.Fatalf("expected error %v, got %v", expected, err)
	}
}

type recordingSlackPoster struct {
	channel string
	err     error
	calls   int
}

func (poster *recordingSlackPoster) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	poster.calls++
	poster.channel = channelID
	return "", "", poster.err
}
