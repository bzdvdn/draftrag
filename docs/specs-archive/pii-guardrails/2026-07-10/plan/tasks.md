# PII Guardrails — Задачи

## Phase Contract

Inputs: `plan.md`, `spec.md`, `data-model.md`.
Outputs: `tasks.md`.
Stop if: AC-* нельзя привязать к исполнимым задачам — нет.

## Surface Map

| Surface | Tasks |
|---------|-------|
| `internal/domain/pii.go` [new] | T1.1 |
| `internal/infrastructure/piidetector/` [new] | T1.2 |
| `pkg/draftrag/pii.go` [new] | T2.1 |
| `pkg/draftrag/draftrag.go` | T2.1, T2.2, T2.3, T3.1 |
| `internal/application/pipeline.go` | T2.1 |
| `pkg/draftrag/search_routing.go` | T3.2 |
| `internal/domain/redaction.go` | T1.1 (reuse) |
| `examples/pii-guardrails/main.go` [new] | T2.5 |
| `internal/infrastructure/piidetector/piidetector_test.go` [new] | T2.4, T3.4, T4.1, T5.1 |
| `internal/infrastructure/piidetector/phone.go` | T5.1 |
| `pkg/draftrag/pii_test.go` | T2.4, T3.4, T4.1, T5.2 |

## Implementation Context

- Цель MVP: PIIDetector interface + 3 built-in detectors (email, phone, SSN) + Index/Query redaction + example
- Инварианты: PIIDetector живёт на публичном слое (DEC-001); CompositePIIDetector композирует под-детекторы (DEC-002); RewrittenQuery redaction — пост-фактум через SearchBuilder (DEC-003)
- Ошибки: nil PIIDetector = no-op; пустой текст = no-op; детектор не возвращает ошибки, всегда возвращает string
- Контракты: `domain.PIIDetector{Detect(text string) string}`; `PIICategories{Email,Phone,SSN,CreditCard bool}`; маркер замены — `<redacted>` (redaction.go)
- Proof signals: `go test ./...` проходит; пример выводит `<redacted>`; ни один существующий тест не сломан
- Вне scope: streaming, metadata, ML-based, IP/даты рождения

## Фаза 1: Основа

Цель: domain-интерфейс + infrastructure-детекторы.

- [x] T1.1 Создать `internal/domain/pii.go` с интерфейсом `PIIDetector`. Touches: `internal/domain/pii.go`, `internal/domain/redaction.go`
- [x] T1.2 Создать `internal/infrastructure/piidetector/` с `EmailDetector`, `PhoneDetector`, `SSNDetector` (regexp, `\b` границы) и `CompositePIIDetector`. Каждый детектор реализует `domain.PIIDetector`. Touches: `internal/infrastructure/piidetector/`

## Фаза 2: MVP Slice

Цель: PipelineOption + Index/Query redaction + пример.

- [x] T2.1 Добавить `PIIDetector` поле в `PipelineOptions` (оба слоя), wire в `NewPipelineWithOptions`, re-export в `pkg/draftrag/pii.go`. Touches: `pkg/draftrag/draftrag.go`, `pkg/draftrag/pii.go`, `internal/application/pipeline.go`
- [x] T2.2 Реализовать PII-redaction в `pkg.Index` — применить `p.piidetector.Detect` к `doc.Content` перед `core.Index`. Nil-безопасно. Touches: `pkg/draftrag/draftrag.go`
- [x] T2.3 Реализовать PII-redaction в `pkg.Query` / `pkg.Retrieve` — применить детектор к `RetrievalResult.Content` после `core`. Nil-безопасно. Touches: `pkg/draftrag/draftrag.go`
- [x] T2.4 Добавить unit-тесты: 3+ кейса на каждый pattern-детектор (SC-002); integration-тесты для AC-001, AC-002, AC-004, AC-006. Touches: `internal/infrastructure/piidetector/piidetector_test.go`, `pkg/draftrag/pii_test.go`
- [x] T2.5 Создать `examples/pii-guardrails/main.go` — демонстрация PII-redaction с InMemoryStore. Touches: `examples/pii-guardrails/main.go`

## Фаза 3: Основная реализация

Цель: Answer redaction, RewrittenQuery, кредитные карты, кастомный детектор.

- [x] T3.1 Реализовать PII-redaction в `pkg.Answer` — применить детектор к `RetrievalResult.Content` в ответе. Nil-безопасно. Touches: `pkg/draftrag/draftrag.go` (redaction через Cite/InlineCite, T2.3)
- [x] T3.2 Реализовать PII-redaction в `RewrittenQuery` — в `routeRewriter` после генерации rewritten queries применить детектор. Touches: `pkg/draftrag/search_routing.go`
- [x] T3.3 Добавить `CreditCardDetector` и включить в `NewDefaultPIIDetector`. Touches: `internal/infrastructure/piidetector/`
- [x] T3.4 Добавить тест кастомного детектора (AC-005): реализовать простой детектор в тесте, подключить к Pipeline, проверить Index. Touches: `pkg/draftrag/pii_test.go`
- [x] T4.1 Добавить тесты для AC-003, AC-005, AC-007 (интеграционные, Pipeline + InMemoryStore). Touches: `pkg/draftrag/pii_test.go`
- [x] T4.2 Добавить benchmark SC-001 и проверить `go vet ./...` + `golangci-lint`. Touches: `internal/infrastructure/piidetector/piidetector_test.go`, `Makefile`

## Фаза 5: Fix verify concerns

Цель: исправить 2 concerns из verify — phone-детектор для РФ-форматов + прямой тест AC-007.

- [x] T5.1 Расширить phone-детектор: добавить паттерн для форматов с короткими сегментами (`+7-900-123-45-67`, `+44-20-7946-0958`). Touches: `internal/infrastructure/piidetector/phone.go`, `internal/infrastructure/piidetector/piidetector_test.go`
- [x] T5.2 Добавить изолированный тест AC-007: Pipeline с QueryRewriter + PIIDetector, проверить что RewrittenQuery не содержит PII. Touches: `pkg/draftrag/pii_test.go`

## Покрытие критериев приемки

- AC-001 -> T2.2, T2.4
- AC-002 -> T2.3, T2.4
- AC-003 -> T3.1, T4.1
- AC-004 -> T1.2, T2.2, T2.4
- AC-005 -> T1.1, T3.4, T4.1
- AC-006 -> T2.2, T2.3, T2.4
- AC-007 -> T3.2, T4.1, T5.2

## Заметки

- Ни один существующий файл не переписывается — только добавление полей в структуры
- Nil PIIDetector = no-op гарантирует backward compatibility
- Пример `examples/pii-guardrails/` следует паттерну `examples/memory/`
