package utils

import (
	"fmt"
	"time"
)

// DurationToString converts a time.Duration to a human-readable string. Respects minutes, hours and days.
func DurationToString(duration time.Duration) string {
	// For a duration less than a day
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		mins := int(duration.Minutes()) % 60

		switch hours {
		case 0:
			return fmt.Sprintf("%d minutes", mins)
		case 1:
			if mins == 0 {
				return "1 hour"
			}
			return fmt.Sprintf("1 hour and %d minutes", mins)
		default:
			if mins == 0 {
				return fmt.Sprintf("%d hours", hours)
			}
			return fmt.Sprintf("%d hours and %d minutes", hours, mins)
		}
	} else {
		// For durations of a day or more
		days := int(duration.Hours() / 24)
		hours := int(duration.Hours()) % 24

		switch hours {
		case 0:
			if days == 1 {
				return "1 day"
			}
			return fmt.Sprintf("%d days", days)
		case 1:
			if days == 1 {
				return "1 day and 1 hour"
			}
			return fmt.Sprintf("%d days and 1 hour", days)
		default:
			if days == 1 {
				return fmt.Sprintf("1 day and %d hours", hours)
			}
			return fmt.Sprintf("%d days and %d hours", days, hours)
		}
	}
}
