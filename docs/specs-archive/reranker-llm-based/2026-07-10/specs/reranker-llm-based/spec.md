# LLM-as-judge reranker (zero-shot, без fine-tune)

## Scope Snapshot

- In scope: компонент переранжирования результатов retrieval, который использует LLM как judge для zero-shot скоринга релевантности каждого чанка к запросу.
- Out of scope: fine-tune ранжирующей модели, cross-encoder reranker, интеграция внешних rerank API (Cohere, etc.).

## Цель

Разработчик RAG-пайплайна получает возможность улучшить качество ранжирования retrieval-результатов за счёт LLM-based zero-shot оценки, без необходимости дообучать модель. Успех фичи измеряется тем, что пользователь может подключить LLM-reranker через `PipelineOptions.Reranker` и наблюдать переранжированные результаты с score от LLM.

## Основной сценарий

1. Пользователь создаёт LLMReranker с LLMProvider и опциональным шаблоном промпта.
2. После retrieval Pipeline вызывает reranker.Rerank с исходным запросом и списком чанков.
3. Reranker формирует batch промпт: запрос + несколько чанков — LLM оценивает релевантность каждого (0–1).
4. Reranker сортирует чанки по убыванию LLM-score и возвращает результат.
5. Если LLM недоступен или возвращает ошибку — reranker возвращает исходный порядок, логируя ошибку.

## User Stories

- **P1 Story**: Как разработчик RAG, я хочу подключить LLM-reranker к Pipeline, чтобы чанки переранжировались по LLM-оценке релевантности, и видеть score в RetrievedChunk.
- **P2 Story**: Как разработчик, я хочу настроить промпт judge (системный промпт + инструкцию скоринга), чтобы адаптировать критерии релевантности под свою доменную область.

## MVP Slice

LLMReranker, реализующий `domain.Reranker` с batch-скорингом чанков через LLM, конфигурируемым промптом и graceful degradation при ошибке LLM.

Обязательные AC: AC-001, AC-002, AC-004.

## First Deployable Outcome

После первого implementation pass можно:
- Создать LLMReranker через конструктор `NewLLMReranker(llm domain.LLMProvider, opts ...) (*LLMReranker, error)`.
- Подключить к Pipeline через `PipelineOptions{Reranker: myReranker}`.
- Выполнить `pipeline.Query(ctx, question)` и увидеть чанки, отсортированные по LLM-score.

## Scope

- Реализация `domain.Reranker` в `internal/infrastructure/reranker/`.
- Конструктор `NewLLMReranker` и тип `LLMReranker` в `pkg/draftrag/reranker_llm.go`.
- Рекспорт `LLMReranker` и опций через публичный пакет `pkg/draftrag`.
- Scorig промпт для LLM с zero-shot инструкцией (системный промпт + формат вывода).
- Batching: несколько чанков в одном LLM-вызове (в пределах лимита контекста).
- Graceful degradation: при ошибке LLM возвращать исходный порядок и логировать.
- Интеграция с `PipelineOptions.Reranker` (уже существует, wiring в `pipeline.go`).

## Контекст

- В репозитории уже существует интерфейс `domain.Reranker` и `domain.BatchReranker`.
- Pipeline вызывает reranker через `maybeRerank` / `maybeRerankBatch` в `retrieval.go`.
- Существующая реализация cross-encoder reranker (`reranker-cross-encoder`) задаёт структурный precedent для нового reranker.
- Ограничение: LLM имеет лимит контекстного окна — batch размер должен учитывать длину промпта + чанков.
- LLMProvider уже доступен как зависимость Pipeline; переиспользование того же provider для reranker — ожидаемый сценарий.

## Зависимости

- `domain.Reranker` и `domain.BatchReranker` — существующие интерфейсы.
- `domain.LLMProvider` — для scoring через LLM.
- `domain.UsageAwareLLMProvider` — опционально для логирования token usage.
- Внешних сервисов/библиотек нет (используется существующий LLMProvider).

## Требования

- RQ-001 Система ДОЛЖНА оценивать релевантность каждого чанка относительно запроса через zero-shot LLM-вызов.
- RQ-002 Система ДОЛЖНА возвращать чанки, отсортированные по убыванию LLM-score.
- RQ-003 Шаблон промпта для judge ДОЛЖЕН быть конфигурируемым через опции конструктора.
- RQ-004 При ошибке LLM-вызова система ДОЛЖНА возвращать исходный порядок чанков (graceful degradation).
- RQ-005 Система ДОЛЖНА поддерживать группировку чанков в batch для минимизации количества LLM-вызовов.
- RQ-006 Система ДОЛЖНА выполнять повторные попытки (retry) LLM-вызова с настраиваемым лимитом (maxRetries) при временных ошибках, прежде чем перейти к graceful degradation.
- RQ-007 Система ДОЛЖНА поддерживать domain.BatchReranker для эффективного multi-query режима.

## Вне scope

- Cross-encoder reranker (BERT-based pairwise scoring).
- Fine-tune ранжирующей модели (LoRA, full fine-tune).
- Интеграция Cohere Rerank API или аналогичных внешних rerank-сервисов.
- Кэширование LLM-оценок (может быть отдельной фичей).
- Поддержка DocumentStore/TransactionalDocumentStore (reranker не пишет в store).
- Автоматический подбор batch size под контекстное окно модели (hardcoded default).

## Критерии приемки

### AC-001 LLM scoring каждого чанка

- Почему это важно: без оценки релевантности reranker не может повлиять на порядок результатов.
- **Given** запрос пользователя "query" и список из N чанков
- **When** reranker.Rerank(ctx, "query", chunks) вызывается
- **Then** каждый чанк в возвращаемом списке имеет Score, установленный LLM (в диапазоне [0,1] или ином документированном диапазоне)
- Evidence: все чанки после rerank имеют Score != исходного score (если LLM изменил порядок), и Score различаются между релевантными и нерелевантными чанками

### AC-002 Переранжирование по убыванию LLM-score

- Почему это важно: цель reranker — улучшить порядок retrieval-результатов.
- **Given** чанки с LLM-оценками [0.9, 0.3, 0.7]
- **When** reranker.Rerank завершился
- **Then** чанки отсортированы по Score убыванию: [0.9, 0.7, 0.3]
- Evidence: тест проверяет порядок чанков после rerank

### AC-003 Конфигурируемый шаблон промпта

- Почему это важно: в разных доменах нужны разные критерии релевантности.
- **Given** пользователь передал кастомный promptTemplate в опциях конструктора
- **When** reranker выполняет LLM-вызов
- **Then** systemPrompt содержит кастомизированный шаблон
- Evidence: тест с mock LLMProvider проверяет, что переданный systemPrompt содержит кастомный текст

### AC-004 Graceful degradation при ошибке LLM

- Почему это важно: отказ reranker не должен ломать retrieval.
- **Given** LLMProvider возвращает ошибку на Generate
- **When** reranker.Rerank вызывается
- **Then** возвращаются исходные чанки (с оригинальными score и порядком), ошибка не возвращается
- Evidence: тест с LLMProvider, возвращающим ошибку, проверяет что result.Chunks == originalChunks (по составу и score)

### AC-005 Batch-скоринг в одном LLM-вызове

- Почему это важно: снижение latency и cost при большом количестве чанков.
- **Given** N чанков для скоринга
- **When** reranker.Rerank выполняется
- **Then** количество LLM-вызовов = ceil(N / batchSize), а не N отдельных вызовов
- Evidence: mock LLMProvider счётчик вызовов после rerank == 1 (при batchSize >= N)

### AC-006 BatchReranker capability

- Почему это важно: multi-query режим Pipeline ожидает BatchReranker для эффективности.
- **Given** LLMReranker реализует domain.BatchReranker
- **When** pipeline в multi-query режиме вызывает maybeRerankBatch
- **Then** chunks переранжированы для каждого query с использованием batch LLM-вызова
- Evidence: type assertion `batchReranker, ok := reranker.(domain.BatchReranker)` успешен; тест проверяет RerankBatch с несколькими query

### AC-007 Retry при временной ошибке LLM

- Почему это важно: временные ошибки LLM не должны приводить к graceful degradation (потере ранжирования), если повторный вызов успешен.
- **Given** LLMProvider возвращает временную ошибку (5xx, network error)
- **When** reranker.Rerank выполняется с maxRetries=2
- **Then** reranker выполняет до maxRetries повторных попыток; при успехе одной из них результат используется
- Evidence: mock LLMProvider счётчик вызовов == maxRetries+1 (первый + retry) при успехе на последней попытке; при исчерпании retry — graceful degradation (AC-004)

## Допущения

- LLMProvider способен обрабатывать до `batchSize` чанков в одном вызове (умещаются в контекст).
- LLM корректно следует формату вывода score в промпте.
- Default batch size (10) разумен для большинства моделей; пользователь может изменить.
- Пользователь не ожидает детерминированного порядка при равных score (sort stable не гарантирован).

## Критерии успеха

- SC-001 LLM-reranker не должен добавлять >500ms latency на batch из 10 чанков (относительно baseline retrieval latency).

## Краевые случаи

- Пустой список чанков: rerank возвращает пустой список без LLM-вызова.
- Один чанк: rerank возвращает его с score от LLM (один LLM-вызов).
- Все чанки получили score = 0: порядок сохраняется исходный.
- Часть чанков не попала в batch из-за ограничения batchSize: обрабатываются в несколько LLM-вызовов.
- LLM вернул непарсимый ответ (не JSON, не число): чанк получает score=0, ошибка логируется.

## Открытые вопросы

- `none` — вопросы прояснены: maxRetries добавлен (RQ-006, AC-007), weighted score fusion отложен (не MVP), шкала 0–10 integer JSON.
- Если weighted score fusion потребуется — оформить отдельной фичей с ясным use case.
