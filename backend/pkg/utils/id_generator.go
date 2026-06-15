package utils

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

func GenerateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func GenerateShortID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func GenerateUserID(prefix string) string {
	return prefix + "_" + GenerateShortID()
}

func NowUnixMilli() int64 {
	return time.Now().UnixMilli()
}

func NowUnix() int64 {
	return time.Now().Unix()
}

func FormatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}
