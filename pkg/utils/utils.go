package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/VictoriaMetrics/metricsql"
)

const (
	varInterval = "$__interval"
)

var (
	defaultResolution int64 = 1500
	year                    = time.Hour * 24 * 365
	day                     = time.Hour * 24
)

const (
	// These values prevent from overflow when storing msec-precision time in int64.
	minTimeMsecs = 0 // use 0 instead of `int64(-1<<63) / 1e6` because the storage engine doesn't actually support negative time
	maxTimeMsecs = int64(1<<63-1) / 1e6
)

// GetTime  returns time from the given string.
func GetTime(s string) (time.Time, error) {
	secs, err := ParseTime(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("cannot parse %s: %w", s, err)
	}
	msecs := int64(secs * 1e3)
	if msecs < minTimeMsecs {
		msecs = 0
	}
	if msecs > maxTimeMsecs {
		msecs = maxTimeMsecs
	}

	return time.Unix(0, msecs*int64(time.Millisecond)).UTC(), nil
}

// ParseTime parses time s in different formats.
//
// See https://docs.victoriametrics.com/Single-server-VictoriaMetrics.html#timestamp-formats
//
// It returns unix timestamp in seconds.
func ParseTime(s string) (float64, error) {
	currentTimestamp := float64(time.Now().UnixNano()) / 1e9
	return ParseTimeAt(s, currentTimestamp)
}

const (
	// time.UnixNano can only store maxInt64, which is 2262
	maxValidYear = 2262
	minValidYear = 1970
)

// ParseTimeAt parses time s in different formats, assuming the given currentTimestamp.
//
// See https://docs.victoriametrics.com/Single-server-VictoriaMetrics.html#timestamp-formats
//
// It returns unix timestamp in seconds.
func ParseTimeAt(s string, currentTimestamp float64) (float64, error) {
	if s == "now" {
		return currentTimestamp, nil
	}
	sOrig := s
	tzOffset := float64(0)
	if len(sOrig) > 6 {
		// Try parsing timezone offset
		tz := sOrig[len(sOrig)-6:]
		if (tz[0] == '-' || tz[0] == '+') && tz[3] == ':' {
			isPlus := tz[0] == '+'
			hour, err := strconv.ParseUint(tz[1:3], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("cannot parse hour from timezone offset %q: %w", tz, err)
			}
			minute, err := strconv.ParseUint(tz[4:], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("cannot parse minute from timezone offset %q: %w", tz, err)
			}
			tzOffset = float64(hour*3600 + minute*60)
			if isPlus {
				tzOffset = -tzOffset
			}
			s = sOrig[:len(sOrig)-6]
		}
	}
	s = strings.TrimSuffix(s, "Z")
	if len(s) > 0 && (s[len(s)-1] > '9' || s[0] == '-') || strings.HasPrefix(s, "now") {
		// Parse duration relative to the current time
		s = strings.TrimPrefix(s, "now")
		d, err := ParseDuration(s)
		if err != nil {
			return 0, err
		}
		if d > 0 {
			d = -d
		}
		return currentTimestamp + float64(d)/1e9, nil
	}
	if len(s) == 4 {
		// Parse YYYY
		t, err := time.Parse("2006", s)
		if err != nil {
			return 0, err
		}
		y := t.Year()
		if y > maxValidYear || y < minValidYear {
			return 0, fmt.Errorf("cannot parse year from %q: year must in range [%d, %d]", s, minValidYear, maxValidYear)
		}
		return tzOffset + float64(t.UnixNano())/1e9, nil
	}
	if !strings.Contains(sOrig, "-") {
		// Parse the timestamp in seconds or in milliseconds
		ts, err := strconv.ParseFloat(sOrig, 64)
		if err != nil {
			return 0, err
		}
		if ts >= (1 << 32) {
			// The timestamp is in milliseconds. Convert it to seconds.
			ts /= 1000
		}
		return ts, nil
	}
	if len(s) == 7 {
		// Parse YYYY-MM
		t, err := time.Parse("2006-01", s)
		if err != nil {
			return 0, err
		}
		return tzOffset + float64(t.UnixNano())/1e9, nil
	}
	if len(s) == 10 {
		// Parse YYYY-MM-DD
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return 0, err
		}
		return tzOffset + float64(t.UnixNano())/1e9, nil
	}
	if len(s) == 13 {
		// Parse YYYY-MM-DDTHH
		t, err := time.Parse("2006-01-02T15", s)
		if err != nil {
			return 0, err
		}
		return tzOffset + float64(t.UnixNano())/1e9, nil
	}
	if len(s) == 16 {
		// Parse YYYY-MM-DDTHH:MM
		t, err := time.Parse("2006-01-02T15:04", s)
		if err != nil {
			return 0, err
		}
		return tzOffset + float64(t.UnixNano())/1e9, nil
	}
	if len(s) == 19 {
		// Parse YYYY-MM-DDTHH:MM:SS
		t, err := time.Parse("2006-01-02T15:04:05", s)
		if err != nil {
			return 0, err
		}
		return tzOffset + float64(t.UnixNano())/1e9, nil
	}
	// Parse RFC3339
	t, err := time.Parse(time.RFC3339, sOrig)
	if err != nil {
		return 0, err
	}
	return float64(t.UnixNano()) / 1e9, nil
}

// ParseDuration parses duration string in Prometheus format
func ParseDuration(s string) (time.Duration, error) {
	ms, err := metricsql.DurationValue(s, 0)
	if err != nil {
		return 0, err
	}
	return time.Duration(ms) * time.Millisecond, nil
}

// ReplaceTemplateVariable get query and use it expression to remove grafana template variables with
func ReplaceTemplateVariable(expr string, interval int64) string {
	expr = strings.ReplaceAll(expr, varInterval, formatDuration(time.Duration(interval)*time.Millisecond))
	return expr
}

func formatDuration(inter time.Duration) string {
	switch {
	case inter >= year:
		return fmt.Sprintf("%dy", inter/year)
	case inter >= day:
		return fmt.Sprintf("%dd", inter/day)
	case inter >= time.Hour:
		return fmt.Sprintf("%dh", inter/time.Hour)
	case inter >= time.Minute:
		return fmt.Sprintf("%dm", inter/time.Minute)
	case inter >= time.Second:
		return fmt.Sprintf("%ds", inter/time.Second)
	case inter >= time.Millisecond:
		return fmt.Sprintf("%dms", inter/time.Millisecond)
	default:
		return "1ms"
	}
}
