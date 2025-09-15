package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Timestamp string

func (t *Timestamp) MarshalJSON() ([]byte, error) {
	ts, err := time.Parse(time.RFC3339, string(*t))
	if err != nil {
		return nil, err
	}
	stamp := fmt.Sprint(ts.Unix())

	return []byte(stamp), nil
}

func (t *Timestamp) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))

	// null -> empty
	if s == "null" || s == "" {
		*t = ""
		return nil
	}

	// Quoted string: could be ISO8601 or numeric inside quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		inner := s[1 : len(s)-1]

		// Try parsing as RFC3339
		if ts, err := time.Parse(time.RFC3339, inner); err == nil {
			*t = Timestamp(ts.In(time.UTC).Format(time.RFC3339))
			return nil
		}

		// Try numeric inside quotes
		if n, err := strconv.Atoi(inner); err == nil {
			*t = Timestamp(time.Unix(int64(n), 0).In(time.UTC).Format(time.RFC3339))
			return nil
		}

		return fmt.Errorf("unsupported timestamp format: %s", inner)
	}

	// Unquoted: try numeric seconds
	if n, err := strconv.Atoi(s); err == nil {
		*t = Timestamp(time.Unix(int64(n), 0).In(time.UTC).Format(time.RFC3339))
		return nil
	}

	// As a fallback, try parsing as RFC3339 without quotes
	if ts, err := time.Parse(time.RFC3339, s); err == nil {
		*t = Timestamp(ts.In(time.UTC).Format(time.RFC3339))
		return nil
	}

	return fmt.Errorf("unsupported timestamp format: %s", s)
}

type ConfigBool bool

func (t *ConfigBool) MarshalJSON() ([]byte, error) {
	var v string
	if *t {
		v = `"1"`
	} else {
		v = `"0"`
	}

	return []byte(v), nil
}

func (t *ConfigBool) UnmarshalJSON(b []byte) error {
	asString := string(b)
	if asString == `"1"` {
		*t = true
	} else {
		*t = false
	}

	return nil
}
