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

		if hours == 0 {
			return fmt.Sprintf("%d minutes", mins)
		} else if mins == 0 {
			if hours == 1 {
				return "1 hour"
			}
			return fmt.Sprintf("%d hours", hours)
		} else {
			if hours == 1 {
				return fmt.Sprintf("1 hour and %d minutes", mins)
			}
			return fmt.Sprintf("%d hours and %d minutes", hours, mins)
		}
	} else {
		// For durations of a day or more
		days := int(duration.Hours() / 24)
		hours := int(duration.Hours()) % 24

		if hours == 0 {
			if days == 1 {
				return "1 day"
			}
			return fmt.Sprintf("%d days", days)
		} else {
			if days == 1 {
				if hours == 1 {
					return "1 day and 1 hour"
				}
				return fmt.Sprintf("1 day and %d hours", hours)
			}
			if hours == 1 {
				return fmt.Sprintf("%d days and 1 hour", days)
			}
			return fmt.Sprintf("%d days and %d hours", days, hours)
		}
	}
}
