package since

import (
	"fmt"
	"time"
)

func Since(t time.Time) string {
	s := time.Since(t)
	out := ""

	years := int(s.Hours() / (24 * 365.25))
	months := int(s.Hours()/(24*30.4375)) % 12
	weeks := int(s.Hours()/(24*7)) % 4
	days := int(s.Hours()/24) % 7
	hours := int(s.Hours()) % 24
	minutes := int(s.Minutes()) % 60
	seconds := int(s.Seconds()) % 60

	if years > 0 {
		out += fmt.Sprintf("%d years", years)
	}
	if months > 0 {
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("%d months", months)
	}
	if weeks > 0 {
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("%d weeks", weeks)
	}
	if days > 0 {
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("%d days", days)
	}
	if hours > 0 {
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("%d hours", hours)
	}
	if minutes > 0 {
		if out != "" {
			out += ", "
		}
		out += fmt.Sprintf("%d minutes", minutes)
	}
	if seconds > 0 && out == "" {
		out = fmt.Sprintf("%d seconds", seconds)
	}

	return out + " ago"
}
