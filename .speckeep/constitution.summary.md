## Purpose

draftRAG — Go-библиотека для быстрого создания RAG-систем. Предоставляет абстракции над векторными хранилищами и LLM-провайдерами.

## Key Constraints

- Только Go 1.21+, нет bindings для других языков
- Нет встроенного HTTP-сервера или CLI — только библиотека
- Все внешние зависимости через Go-интерфейсы
- Clean Architecture: domain → application → infrastructure
- Контекст (`context.Context`) во всех публичных операциях

## Language Policy

- docs: русский
- agent: русский
- comments: русский

## Development Workflow

- feature-ветки: `feature/<slug>`
- spec перед кодом: `.draftspec/specs/<slug>/spec.md`
- plan и tasks из spec: `.draftspec/plans/<slug>/`
- `go vet`, `go fmt`, `golangci-lint` без ошибок
- Unit-тесты для всех новых функций

## Decision Priorities

- Простота > расширяемость (если не нужна мульти-провайдерность)
- Корректность > скорость
- Поддерживаемость > cleverness
- Интерфейсы > конкретные типы (для внешних зависимостей)
