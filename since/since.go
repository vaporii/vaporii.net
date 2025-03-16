package since

import (
	"fmt"
	"time"
)

func pluralize(value int, singular, plural string) string {
	if value == 1 {
		return fmt.Sprintf("%d %s", value, singular)
	}
	return fmt.Sprintf("%d %s", value, plural)
}

func Since(t time.Time) string {
	s := time.Since(t)
	if s.Seconds() <= 5 {
		return "now"
	}

	out := ""

	years := int(s.Hours() / (24 * 365.25))
	months := int(s.Hours()/(24*30.4375)) % 12
	weeks := int(s.Hours()/(24*7)) % 4
	days := int(s.Hours()/24) % 7
	hours := int(s.Hours()) % 24
	minutes := int(s.Minutes()) % 60
	seconds := int(s.Seconds()) % 60

	if years > 0 {
		out += pluralize(years, "year", "years")
	}
	if months > 0 {
		if out != "" {
			out += ", "
		}
		out += pluralize(months, "month", "months")
	}
	if weeks > 0 {
		if out != "" {
			out += ", "
		}
		out += pluralize(weeks, "week", "weeks")
	}
	if days > 0 {
		if out != "" {
			out += ", "
		}
		out += pluralize(days, "day", "days")
	}
	if hours > 0 {
		if out != "" {
			out += ", "
		}
		out += pluralize(hours, "hour", "hours")
	}
	if minutes > 0 {
		if out != "" {
			out += ", "
		}
		out += pluralize(minutes, "minute", "minutes")
	}
	if seconds > 0 && out == "" {
		out = pluralize(seconds, "second", "seconds")
	}

	return out + " ago"
}
