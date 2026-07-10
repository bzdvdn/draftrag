package piidetector

import (
	"testing"
)

// @sk-test pii-guardrails#T2.4: TestEmailDetector (SC-002, AC-004)
func TestEmailDetector(t *testing.T) {
	d := NewEmailDetector()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain email", "contact: user@example.com", "contact: <redacted>"},
		{"no email", "hello world", "hello world"},
		{"email with dots", "my.name+tag@sub.domain.co.uk", "<redacted>"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := d.Detect(tc.input)
			if got != tc.want {
				t.Errorf("Detect(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// @sk-test pii-guardrails#T2.4: TestPhoneDetector (SC-002, AC-004)
//
// @sk-test pii-guardrails#T5.1: добавлены РФ/Европа форматы (verify concern #1)
func TestPhoneDetector(t *testing.T) {
	d := NewPhoneDetector()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"E.164", "call +1-555-123-4567 now", "call <redacted> now"},
		{"no phone", "just text", "just text"},
		{"with parens", "tel: +1 (555) 123-4567", "tel: <redacted>"},
		{"RF format", "тел: +7-900-123-45-67", "тел: <redacted>"},
		{"UK format", "tel: +44-20-7946-0958", "tel: <redacted>"},
		{"DE format", "tel: +49-30-123-456-78", "tel: <redacted>"},
		{"RF with parens", "tel: +7 (900) 123-45-67", "tel: <redacted>"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := d.Detect(tc.input)
			if got != tc.want {
				t.Errorf("Detect(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// @sk-test pii-guardrails#T2.4: TestSSNDetector (SC-002, AC-004)
func TestSSNDetector(t *testing.T) {
	d := NewSSNDetector()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"valid SSN", "SSN: 123-45-6789", "SSN: <redacted>"},
		{"no SSN", "hello world", "hello world"},
		{"partial number", "number 123-45 is short", "number 123-45 is short"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := d.Detect(tc.input)
			if got != tc.want {
				t.Errorf("Detect(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// @sk-test pii-guardrails#T3.3: TestCreditCardDetector (SC-002, RQ-003)
func TestCreditCardDetector(t *testing.T) {
	d := NewCreditCardDetector()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain 16 digits", "card: 4111 1111 1111 1111", "card: <redacted>"},
		{"with dashes", "4111-1111-1111-1111", "<redacted>"},
		{"no card", "just text", "just text"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := d.Detect(tc.input)
			if got != tc.want {
				t.Errorf("Detect(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// @sk-test pii-guardrails#T2.4: TestCompositeDetector (DEC-002, AC-004)
func TestCompositeDetector(t *testing.T) {
	d := NewDefaultPIIDetector(PIICategories{Email: true, Phone: true, SSN: false})

	input := "email: user@test.com, phone: +1-555-123-4567, ssn: 123-45-6789"
	got := d.Detect(input)

	if contains(got, "user@test.com") {
		t.Error("expected email to be redacted")
	}
	if contains(got, "+1-555-123-4567") {
		t.Error("expected phone to be redacted")
	}
	if !contains(got, "123-45-6789") {
		t.Error("expected SSN to remain (disabled category)")
	}
}

// @sk-test pii-guardrails#T2.4: TestCompositeDetectorEmpty (AC-006)
func TestCompositeDetectorEmpty(t *testing.T) {
	d := NewDefaultPIIDetector(PIICategories{})
	input := "email: user@test.com, phone: +1-555-123-4567"
	got := d.Detect(input)
	if got != input {
		t.Error("expected no change with no categories enabled")
	}
}

// @sk-test pii-guardrails#T4.2: BenchmarkPIIDetectors (SC-001)
func BenchmarkPIIDetectors(b *testing.B) {
	det := NewDefaultPIIDetector(PIICategories{Email: true, Phone: true, SSN: true, CreditCard: true})
	input := "contact: user@example.com, phone: +1-555-123-4567, ssn: 123-45-6789, card: 4111-1111-1111-1111"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		det.Detect(input)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
