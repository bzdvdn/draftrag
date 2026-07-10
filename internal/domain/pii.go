package domain

// @sk-task pii-guardrails#T1.1: PIIDetector interface (RQ-004, AC-005)
//
// PIIDetector определяет интерфейс для обнаружения и цензурирования
// персональных данных (PII) в тексте.
type PIIDetector interface {
	// Detect возвращает текст с заменёнными PII-вхождениями.
	// Если PII не обнаружено, возвращает исходный текст без изменений.
	Detect(text string) string
}
