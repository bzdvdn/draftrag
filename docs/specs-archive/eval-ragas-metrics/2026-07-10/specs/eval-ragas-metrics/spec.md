# RAGAS-style evaluation metrics

## Scope Snapshot

- In scope: добавление трёх RAGAS-совместимых LLM-ассистированных метрик качества RAG-пайплайна: Faithfulness, Answer Relevance, Context Relevance.
- Out of scope: изменение существующих retrieval-метрик (Hit@K, MRR, NDCG, Precision, Recall); гарнитура запуска eval; визуализация; экспорт в сторонние платформы.

## Цель

Разработчик, использующий draftRAG для RAG-пайплайна, получает возможность оценить качество ответов LLM (Faithfulness), релевантность ответа вопросу (Answer Relevance) и релевантность извлечённого контекста запросу (Context Relevance) через те же механизмы eval-пакета. Успех фичи измеряется тем, что пользователь может вызвать три новые метрики из `pkg/draftrag/eval/` и получить нормализованные score [0,1] без разрыва существующего API.

## Основной сценарий

1. Пользователь создаёт eval-датасет с вопросами, контекстами и эталонными ответами.
2. Пользователь запускает eval через существующий `Run` или новый `RunWithAnswer`, который выполняет retrieval и генерацию ответа.
3. Для каждого кейса вычисляются Faithfulness, Answer Relevance, Context Relevance.
4. Результаты агрегируются в отчёт и доступны в `Report.Metrics`.
5. Если LLM-провайдер не указан для LLM-ассистированных метрик — метрика пропускается со score 0, без ошибки.

## User Stories

- P1 Story: разработчик может вычислить Faithfulness для своих RAG-ответов, указав LLMProvider для оценки фактологической согласованности.
- P2 Story: разработчик может вычислить Answer Relevance и Context Relevance как дополнительную диагностику качества.

## MVP Slice

Faithfulness (P1) + базовый прототип Answer Relevance и Context Relevance. AC-001, AC-002, AC-003 обязательны к закрытию.

## First Deployable Outcome

Пакет `pkg/draftrag/eval/` экспортирует три новые функции `ComputeFaithfulness`, `ComputeAnswerRelevance`, `ComputeContextRelevance` и/или интегрирует их в `RunWithAnswer`. Демонстрация в тесте с in-memory retrieval и mock LLM.

## Scope

- Новый тип `RAGASEvaluator` (или набор функций) в `pkg/draftrag/eval/` для LLM-ассистированных метрик.
- Интеграция с существующим `RetrievalRunner` для получения контекста.
- Использование `LLMProvider` и/или `Embedder` из public API draftRAG для LLM-ассистированных вычислений.
- Новая опция `Options` для включения RAGAS-метрик (по аналогии с `EnableNDCG`, `EnablePrecision`, `EnableRecall`).

## Контекст

- draftRAG — библиотека, не сервер; eval — офлайн-утилита, не real-time.
- LLMProvider уже покрыт интерфейсом `UsageAwareLLMProvider` с подсчётом токенов.
- Embedder покрыт интерфейсом `Embedder`.
- Для Faithfulness нужен LLM (декомпозиция ответа на claims + верификация).
- Для Answer Relevance нужен Embedder (косинусная близость между эмбеддингами сгенерированных вопросов и исходного) или LLM (оценка релевантности).
- Для Context Relevance нужен Embedder (релевантность каждого чанка вопросу) — либо бинарная (LLM), либо семантическая (embedder).

## Зависимости

- `pkg/draftrag/eval/harness.go` — существующий eval-пакет.
- `internal/domain/interfaces.go:LLMProvider` (или `UsageAwareLLMProvider`) — для LLM-вызовов.
- `internal/domain/interfaces.go:Embedder` — для эмбеддингов.
- Внешних библиотек не требуется; реализация через существующие интерфейсы.

## Требования

- RQ-001 Система ДОЛЖНА вычислять Faithfulness Score как долю утверждений в ответе, подтверждённых контекстом, через LLM-декомпозицию.
- RQ-002 Система ДОЛЖНА вычислять Answer Relevance Score как семантическую близость между сгенерированными из ответа вопросами и исходным вопросом.
- RQ-003 Система ДОЛЖНА вычислять Context Relevance Score как среднюю релевантность чанков контекста к вопросу.
- RQ-004 Система ДОЛЖНА интегрировать три метрики в отчёт `Metrics` публичного API eval-пакета.
- RQ-005 Система ДОЛЖНА корректно обрабатывать случаи пустого ответа, пустого контекста и nil LLMProvider/Embedder — score 0, без паники.

## Вне scope

- Визуализация метрик (дашборды, графики).
- Экспорт метрик в сторонние системы (LangSmith, MLflow).
- Кастомные prompt-шаблоны для Faithfulness-декомпозиции (используется встроенный дефолтный).
- Поддержка мультиязычных prompt-шаблонов — только английский для internal prompts.
- Асинхронный/батчевый расчёт метрик — синхронный последовательный расчёт.

## Критерии приемки

### AC-001 Faithfulness корректно вычисляется через LLM

- Почему это важно: пользователь должен получить надёжную оценку фактологической согласованности ответа с контекстом.
- **Given** eval-кейс с вопросом, контекстом и ответом и настроенный LLMProvider для Faithfulness
- **When** вызывается `ComputeFaithfulness(ctx, answer, context, llmProvider)`
- **Then** возвращается float64 score в [0,1] и nil error
- Evidence: score = 1.0 для ответа, целиком подтверждённого контекстом; score < 1.0, если часть утверждений не подтверждена

### AC-002 Answer Relevance корректно вычисляется

- Почему это важно: пользователь получает оценку того, насколько ответ релевантен вопросу.
- **Given** eval-кейс с вопросом и ответом и настроенный Embedder
- **When** вызывается `ComputeAnswerRelevance(ctx, question, answer, embedder)`
- **Then** возвращается float64 score в [0,1] и nil error
- Evidence: score выше для прямого ответа на вопрос, чем для нерелевантного ответа

### AC-003 Context Relevance корректно вычисляется

- Почему это важно: пользователь диагностирует перегруженность контекста нерелевантными чанками.
- **Given** eval-кейс с вопросом и набором чанков контекста и настроенный Embedder
- **When** вызывается `ComputeContextRelevance(ctx, question, contextChunks, embedder)`
- **Then** возвращается float64 score в [0,1] и nil error
- Evidence: score = 1.0 когда все чанки релевантны вопросу; score падает с добавлением нерелевантных чанков

### AC-004 Интеграция в отчёт eval-пакета

- Почему это важно: пользователь видит все метрики в одном отчёте без ручного агрегирования.
- **Given** пользовательский код, вызывающий eval.RunWithAnswer с RAGAS-опциями
- **When** в Report.Metrics присутствуют Faithfulness, AnswerRelevance, ContextRelevance
- **Then** поля заполнены средними значениями по всем кейсам
- Evidence: `report.Metrics.Faithfulness != 0` после прогона с кейсом, где ответ полностью подтверждён контекстом

### AC-005 Graceful degradation при nil LLMProvider/Embedder

- Почему это важно: пользователь может запустить eval без LLM для тестового прогона.
- **Given** eval-кейсы с nil полем LLMProvider или Embedder
- **When** запускается eval с включёнными RAGAS-метриками
- **Then** метрики, требующие nil-провайдер, устанавливаются в 0, без ошибки
- Evidence: `report.Metrics.Faithfulness == 0`, `err == nil`

### AC-006 Обработка пустого ответа

- Почему это важно: пограничные случаи не должны вызывать панику.
- **Given** пустой answer (""), непустой контекст
- **When** вызывается `ComputeFaithfulness`
- **Then** возвращается 0 и nil error
- Evidence: no panic, score = 0.0

## Допущения

- LLMProvider, используемый для Faithfulness, может быть тем же или другим, чем для генерации ответа.
- Для Faithfulness используется chain-of-thought декомпозиция на одном LLM-вызове (декомпозиция + верификация в одном запросе).
- Для Answer Relevance: генерируется N вопросов из ответа (N=3 default), усредняется косинусная близость их эмбеддингов с эмбеддингом исходного вопроса.
- Для Context Relevance: вычисляется как средняя косинусная близость эмбеддинга вопроса к эмбеддингу каждого чанка, порог релевантности не применяется.
- RAGAS-метрики опциональны — существующий `Run` без изменений, новый `RunWithAnswer` или отдельные функции.

## Критерии успеха

- SC-001 Вычисление трёх метрик для одного кейса занимает не более 2 LLM-вызовов (один для Faithfulness, один для генерации вопросов в Answer Relevance; Context Relevance без LLM).
- SC-002 Покрытие тестами новых функций ≥70%.

## Краевые случаи

- Пустой контекст (no chunks retrieved) → Faithfulness score = 0, Context Relevance = 0.
- Ответ, не содержащий утверждений (пустой или только стоп-слова) → Faithfulness score = 0.
- Все чанки дублируются → Context Relevance = 1.0 (опрос не штрафует дубликаты).
- LLM-вызов для Faithfulness возвращает ошибку (timeout/rate-limit) → возвращается ошибка, а не 0 (пользователь должен знать о сбое).
- Embedder возвращает ошибку → возвращается ошибка.

## Открытые вопросы

- `none`
