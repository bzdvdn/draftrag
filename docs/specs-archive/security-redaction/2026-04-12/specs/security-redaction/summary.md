---
slug: security-redaction
generated_at: 2026-04-12
---

## Goal

Гарантировать, что известные библиотеке секреты не утекут в ошибки и structured logs, и описать границы ответственности в docs.

## Acceptance Criteria

| ID | Summary | Proof Signal |
|---|---|---|
| AC-001 | Redaction в `err.Error()` провайдеров | Тесты подтверждают отсутствие секрета в ошибке |
| AC-002 | Redaction в structured logs | Logger-тест подтверждает отсутствие секрета в msg/fields |
| AC-003 | Документация про границы redaction | В README/доках есть секция про redaction и ответственность |

## Out of Scope

- Автоматическое PII/DLP обнаружение
- Редакция всех документов/запросов по умолчанию
- Секрет-менеджмент, ротация и хранение ключей

