package utils

import (
	"encoding/base64"
	"time"
)

func EncodeCursor(t time.Time) string {
	return base64.StdEncoding.EncodeToString([]byte(t.Format(time.RFC3339Nano)))
}

func DecodeCursor(encoded string) (time.Time, error) {
	bytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339Nano, string(bytes))
}
