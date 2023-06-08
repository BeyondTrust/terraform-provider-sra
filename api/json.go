package api

import (
	"fmt"
	"strconv"
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
	ts, err := strconv.Atoi(string(b))
	if err != nil {
		return err
	}

	// All timestamps from the server are in UTC
	*t = Timestamp(time.Unix(int64(ts), 0).In(time.UTC).Format(time.RFC3339))

	return nil
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
