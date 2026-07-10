# PII Guardrails — Data Model

## Status: partial change

### Core domain model — no change

- `domain.Document` — без изменений
- `domain.Chunk` — без изменений
- `domain.RetrievalResult` — без изменений
- `domain.RewrittenQuery` — без изменений

### New domain interface

**`domain.PIIDetector`** — новый интерфейс:

```go
type PIIDetector interface {
    // Detect возвращает текст с заменёнными PII-вхождениями.
    // Если PII не обнаружено, возвращает исходный текст без изменений.
    Detect(text string) string
}
```

### New infrastructure types

- `CompositePIIDetector` — применяет набор под-детекторов; конфигурируется категориями
- `EmailDetector`, `PhoneDetector`, `SSNDetector`, `CreditCardDetector` — отдельные regexp-based детекторы

### Configuration model changes

`pkg/draftrag.PipelineOptions`:
- Новое опциональное поле: `PIIDetector domain.PIIDetector`

`application.PipelineOptions`:
- Новое опциональное поле: `PIIDetector domain.PIIDetector`

### Public API additions

`pkg/draftrag/pii.go`:
- `type PIIDetector = domain.PIIDetector`
- `type PIICategories struct { Email, Phone, SSN, CreditCard bool }`
- `func NewDefaultPIIDetector(cats PIICategories) PIIDetector`
- `func NewCompositePIIDetector(detectors ...PIIDetector) PIIDetector`

### No contract changes

- Ни один существующий интерфейс, тип или сигнатура метода не меняется
- Nil PIIDetector → поведение Pipeline не меняется (backward compatible)
