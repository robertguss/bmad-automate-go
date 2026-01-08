package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"1 second", time.Second, "1s"},
		{"30 seconds", 30 * time.Second, "30s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"1 minute", time.Minute, "1m 00s"},
		{"1 minute 30 seconds", 90 * time.Second, "1m 30s"},
		{"5 minutes", 5 * time.Minute, "5m 00s"},
		{"10 minutes 5 seconds", 10*time.Minute + 5*time.Second, "10m 05s"},
		{"1 hour", time.Hour, "60m 00s"},
		{"1 hour 30 minutes", 90 * time.Minute, "90m 00s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDurationExtended(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0s"},
		{"1 second", time.Second, "1s"},
		{"30 seconds", 30 * time.Second, "30s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"1 minute", time.Minute, "1m 00s"},
		{"1 minute 30 seconds", 90 * time.Second, "1m 30s"},
		{"5 minutes", 5 * time.Minute, "5m 00s"},
		{"59 minutes 59 seconds", 59*time.Minute + 59*time.Second, "59m 59s"},
		{"1 hour", time.Hour, "1h 00m"},
		{"1 hour 30 minutes", 90 * time.Minute, "1h 30m"},
		{"2 hours 45 minutes", 2*time.Hour + 45*time.Minute, "2h 45m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDurationExtended(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDurationCompact(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0ms"},
		{"500ms", 500 * time.Millisecond, "500ms"},
		{"999ms", 999 * time.Millisecond, "999ms"},
		{"1 second", time.Second, "1.0s"},
		{"1.5 seconds", 1500 * time.Millisecond, "1.5s"},
		{"30 seconds", 30 * time.Second, "30.0s"},
		{"59 seconds", 59 * time.Second, "59.0s"},
		{"1 minute", time.Minute, "1m0s"},
		{"1 minute 30 seconds", 90 * time.Second, "1m30s"},
		{"5 minutes", 5 * time.Minute, "5m0s"},
		{"59 minutes 59 seconds", 59*time.Minute + 59*time.Second, "59m59s"},
		{"1 hour", time.Hour, "1h0m"},
		{"1 hour 30 minutes", 90 * time.Minute, "1h30m"},
		{"2 hours 45 minutes", 2*time.Hour + 45*time.Minute, "2h45m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDurationCompact(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDurationLong(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"zero", 0, "0 seconds"},
		{"1 second", time.Second, "1 second"},
		{"30 seconds", 30 * time.Second, "30 seconds"},
		{"1 minute", time.Minute, "1 minute"},
		{"1 minute 1 second", 61 * time.Second, "1 minute 1 second"},
		{"1 minute 30 seconds", 90 * time.Second, "1 minute 30 seconds"},
		{"5 minutes", 5 * time.Minute, "5 minutes"},
		{"5 minutes 1 second", 5*time.Minute + time.Second, "5 minutes 1 second"},
		{"10 minutes 5 seconds", 10*time.Minute + 5*time.Second, "10 minutes 5 seconds"},
		{"1 hour", time.Hour, "1 hour"},
		{"1 hour 1 minute", time.Hour + time.Minute, "1 hour 1 minute"},
		{"2 hours 30 minutes", 150 * time.Minute, "2 hours 30 minutes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDurationLong(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}
