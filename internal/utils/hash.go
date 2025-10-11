package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// ComputeHMACSHA256 возвращает код HMAC-SHA256 данных с использованием ключа.
func ComputeHMACSHA256(data []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	sum := mac.Sum(nil)
	return hex.EncodeToString(sum)
}
