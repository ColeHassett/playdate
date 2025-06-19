package internal

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

const timeFormat = "Jan 2 2006 at 03:04 PM"

// FormatTime formats a time.Time object into a human-readable string format.
func FormatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(timeFormat)
}

// RelativeTime formats a given time.Time value into a human-readable string indicating
// how long ago or how long from now it occurred. It breaks down the difference
// into the largest appropriate unit (seconds, minutes, hours, days, months, years).
func RelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t) // if t is in future, diff will be negative

	absDiff := diff
	suffix := "ago"

	if diff < 0 {
		absDiff = t.Sub(now) // Calculate positive duration from now to t
		suffix = "from now"
	}

	switch {
	case absDiff < time.Second:
		return "just now"
	case absDiff < time.Minute:
		seconds := int(absDiff.Seconds())
		if seconds == 1 {
			return fmt.Sprintf("%d second %s", seconds, suffix)
		}
		return fmt.Sprintf("%d seconds %s", seconds, suffix)
	case absDiff < time.Hour:
		minutes := int(absDiff.Minutes())
		if minutes == 1 {
			return fmt.Sprintf("%d minute %s", minutes, suffix)
		}
		return fmt.Sprintf("%d minutes %s", minutes, suffix)
	case absDiff < 24*time.Hour: // Less than 1 day
		hours := int(absDiff.Hours())
		if hours == 1 {
			return fmt.Sprintf("%d hour %s", hours, suffix)
		}
		return fmt.Sprintf("%d hours %s", hours, suffix)
	case absDiff < 30*24*time.Hour: // Less than approx 1 month (using 30 days)
		days := int(absDiff.Hours() / 24)
		if days == 1 {
			return fmt.Sprintf("%d day %s", days, suffix)
		}
		return fmt.Sprintf("%d days %s", days, suffix)
	case absDiff < 365*24*time.Hour: // Less than approx 1 year (using 365 days)
		// For months, a simple division by 30 days is a common approximation for relative time
		months := int(absDiff.Hours() / (30 * 24))
		if months == 1 {
			return fmt.Sprintf("%d month %s", months, suffix)
		}
		return fmt.Sprintf("%d months %s", months, suffix)
	default: // 1 year or more
		// For years, a simple division by 365 days is a common approximation for relative time
		years := int(absDiff.Hours() / (365 * 24))
		if years == 1 {
			return fmt.Sprintf("%d year %s", years, suffix)
		}
		return fmt.Sprintf("%d years %s", years, suffix)
	}
}

// GenerateRandomState generates a URL-safe random string for the 'state' parameter.
func GenerateRandomState() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
