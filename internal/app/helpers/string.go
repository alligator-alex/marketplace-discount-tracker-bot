package helpers

import (
	"math/rand"
	"strings"
	"time"
)

// Concatenate multiple strings into one.
func ConcatStrings(values ...string) string {
	var builder strings.Builder

	for _, value := range values {
		builder.WriteString(value)
	}

	return builder.String()
}

// Get random string with length of 7.
func GetRandomString() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 7

	randomValue := rand.New(rand.NewSource(time.Now().UnixNano()))

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[randomValue.Intn(len(charset))]
	}

	return string(result)
}
