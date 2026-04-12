package domain

import "strings"

const redactedMarker = "<redacted>"

// RedactSecret заменяет все вхождения секрета в тексте на "<redacted>".
// Пустой/пробельный secret → no-op.
func RedactSecret(text, secret string) string {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return text
	}
	return strings.ReplaceAll(text, secret, redactedMarker)
}

// RedactSecrets последовательно применяет RedactSecret для списка секретов.
func RedactSecrets(text string, secrets ...string) string {
	for _, s := range secrets {
		text = RedactSecret(text, s)
	}
	return text
}
