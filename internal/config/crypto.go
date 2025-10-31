package config

import "strings"

const key = "newslettercli"

func Encrypt(input string) string {
	if input == "" {
		return ""
	}
	out := make([]rune, len(input))
	for i, r := range input {
		out[i] = r ^ rune(key[i%len(key)])
	}
	return string(out)
}

func Decrypt(input string) string {
	return Encrypt(input) // XOR is symmetric
}

func Mask(s string) string {
	if len(s) == 0 {
		return "(empty)"
	}
	return strings.Repeat("*", len(s))
}
