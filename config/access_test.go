package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestGetDurationTreatsBareNumbersAsUnit(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	viper.Set(defaultclienttimeout, 20)

	duration := getDuration(defaultclienttimeout, time.Second)
	if duration != 20*time.Second {
		t.Fatalf("expected 20s, got %s", duration)
	}
}

func TestGetDurationAcceptsDurationStrings(t *testing.T) {
	viper.Reset()
	defer viper.Reset()
	viper.Set(defaultclienttimeout, "1500ms")

	duration := getDuration(defaultclienttimeout, time.Second)
	if duration != 1500*time.Millisecond {
		t.Fatalf("expected 1500ms, got %s", duration)
	}
}
