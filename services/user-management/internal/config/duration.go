package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseDuration is like time.ParseDuration but also accepts a whole-number
// "<n>d" (days) or "<n>w" (weeks) value, e.g. "7d" or "2w". Plain Go durations
// such as "168h" or "15m" still work.
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)

	if unit, ok := strings.CutSuffix(s, "d"); ok {
		return daysOrWeeks(unit, 24*time.Hour, s)
	}
	if unit, ok := strings.CutSuffix(s, "w"); ok {
		return daysOrWeeks(unit, 7*24*time.Hour, s)
	}
	return time.ParseDuration(s)
}

func daysOrWeeks(number string, per time.Duration, original string) (time.Duration, error) {
	n, err := strconv.Atoi(number)
	if err != nil {
		// Not a plain "<n>d" — let time.ParseDuration report a normal error.
		return time.ParseDuration(original)
	}
	if n < 0 {
		return 0, fmt.Errorf("negative duration %q", original)
	}
	return time.Duration(n) * per, nil
}
