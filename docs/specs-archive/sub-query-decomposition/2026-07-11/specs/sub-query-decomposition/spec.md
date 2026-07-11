# Sub-query decomposition

## Scope Snapshot

- In scope: разбиение сложного пользовательского запроса на под-вопросы, параллельный retrieval по каждому под-вопросу, объединение контекста и генерация ответа с учётом всех под-вопросов.
- Out of scope: multi-turn диалог с сохранением истории между вызовами; планирование порядка выполнения под-вопросов с зависимостями (DAG execution); агентный loop с обращением к внешним инструментам.

## Цель

Разработчик, использующий draftRAG для question-answering над сложным корпусом, получает более полные и точные ответы на multi-faceted запросы: система самостоятельно разбивает запрос на несколько под-вопросов, собирает релевантный контекст по каждому, объединяет его и генерирует ответ. Успех измеряется покрытием: ответ должен затрагивать все аспекты, идентифицированные в под-вопросах.

## Основной сценарий

1. Пользователь задаёт сложный запрос (напр., «Каковы требования к безопасности и стоимость лицензирования для AWS и Azure?») через `SearchBuilder.SubDecompose()`.
2. Pipeline определяет стратегию декомпозиции: при наличии LLMProvider используется LLM-based генерация под-вопросов; если LLM недоступен или запрос простой (до N слов/одна тема) — rule-based splitting (по союзам/разделителям).
3. Каждый под-вопрос выполняется как отдельный retrieval-запрос (embedding → VectorStore.Search), параллельно с использованием существующего worker pool.
4. Результаты всех под-вопросов объединяются: дедупликация (по ID чанка), реранжирование по max score (каждый чанк сохраняет лучший score среди всех под-вопросов).
5. Объединённый контекст передаётся в `LLMProvider.Generate` с prompt, учитывающим исходный вопрос и все под-вопросы.
6. Сгенерированный ответ возвращается пользователю.
7. При ошибке LLM-декомпозиции pipeline выполняет fallback на rule-based splitting; если rule-based также не дал под-вопросов — выполняется обычный single-query retrieval + answer (graceful degradation). На каждом шаге pipeline логирует причину fallback.

## User Stories

- P1 Story: разработчик подключает готовый дефолтный decomposer (LLM-based) через опцию Pipeline и получает multi-faceted ответ без дополнительного кода.
- P2 Story: разработчик реализует кастомную стратегию декомпозиции через интерфейс и подключает её через PipelineOptions или per-request.

## MVP Slice

LLM-based decomposition с дефолтным prompt + параллельный retrieval + merge + answer. Rule-based стратегия и кастомный интерфейс — P2.

AC-001, AC-002, AC-003, AC-005, AC-007 — обязательны для MVP.

## First Deployable Outcome

Caller может вызвать `pipeline.Search("сложный запрос").SubDecompose().Answer(ctx)` и получить ответ, который покрывает все аспекты запроса, без каких-либо дополнительных настроек. Результат можно сравнить с обычным `Answer` для оценки улучшения.

## Scope

- Новый интерфейс `QueryDecomposer` в domain: метод `Decompose(ctx, query string) ([]string, error)`
- Две built-in реализации: `LLMQueryDecomposer` (LLM-based, через LLMProvider) и `RuleQueryDecomposer` (rule-based splitting)
- Новый метод `SubDecompose()` на `SearchBuilder` — включает декомпозицию для данного запроса
- Новое поле `QueryDecomposer` в `PipelineOptions` — pipeline-level decomposer (nil = отключено)
- Выбор стратегии: pipeline-level decomposer может быть композитным (LLM → fallback на rule-based)
- Параллельный retrieval по под-вопросам: reuse `processDocsConcurrently` или аналогичного worker pool
- Merge-логика: объединение RetrievalResult, дедупликация по Chunk.ID, max score per chunk
- Поддержка в `Retrieve` / `Answer` / `Cite` / `InlineCite` / `Stream` / `StreamSources` / `StreamCite` путём добавления routing-хендлера для sub-decompose режима
- Перенос источников (sources) для `Cite` и `InlineCite` (все под-вопросы → один merged список источников)

## Контекст

- Существующий интерфейс `QueryRewriter` уже поддерживает 1:N режим (multi-query). Sub-query decomposition — семантически другой паттерн: под-вопросы не заменяют запрос, а декомпозируют его на независимые аспекты. Поэтому нужен отдельный интерфейс, не переусложняя QueryRewriter.
- Параллельный retrieval уже частично реализован в `MultiQuery` — можно использовать похожий подход.
- Все публичные операции принимают `context.Context` — cancellation распространяется на все под-запросы.
- Интеграция с hooks: `OnStart`/`OnEnd` для всего sub-decompose цикла + для каждого под-вопроса.

## Зависимости

- Зависит от `LLMProvider` (для LLM-based decomposition) — nil должен приводить к graceful fallback на rule-based или single-query.
- Reuse существующего `VectorStore` для retrieval по под-вопросам.
- Reuse существующего `Embedder` для embedding под-вопросов.
- `none` внешних сервисных зависимостей сверх уже имеющихся.

## Требования

- RQ-001 Pipeline ДОЛЖЕН поддерживать разбиение сложного запроса на под-вопросы через `QueryDecomposer`.
- RQ-002 Pipeline ДОЛЖЕН выполнять retrieval по каждому под-вопросу параллельно с использованием worker pool.
- RQ-003 Pipeline ДОЛЖЕН объединять результаты retrieval всех под-вопросов: дедуплицировать чанки по Chunk.ID, сохранять максимальный score для каждого уникального чанка.
- RQ-004 Pipeline ДОЛЖЕН генерировать ответ на исходный запрос с использованием объединённого контекста из всех под-вопросов.
- RQ-005 Pipeline ДОЛЖЕН поддерживать graceful degradation: при ошибке декомпозиции или пустом списке под-вопросов — выполнить обычный single-query retrieve + answer.
- RQ-006 Pipeline ДОЛЖЕН поддерживать per-request включение sub-decompose через `SearchBuilder.SubDecompose()`.
- RQ-007 `QueryDecomposer` ДОЛЖЕН быть опциональным (nil в PipelineOptions = отключено).

## Вне scope

- Агентный loop: итеративное уточнение под-вопросов на основе промежуточных результатов retrieval.
- DAG execution: под-вопросы с зависимостями (результат одного под-вопроса влияет на формулировку другого).
- Multi-turn: сохранение истории декомпозиции между вызовами в рамках диалога.
- Персистентное кэширование результатов декомпозиции.
- Визуализация/экспорт под-вопросов для отладки (может быть добавлено через hooks).
- Per-sub-question reranking (применяется один reranker на merged результат, как и в обычном режиме).

## Критерии приемки

### AC-001 Sub-decompose включён в SearchBuilder

- Почему это важно: пользователь должен иметь возможность включить декомпозицию для конкретного запроса без создания нового Pipeline.
- **Given** Pipeline с LLMProvider и VectorStore
- **When** вызывается `pipeline.Search("сложный запрос с двумя аспектами").SubDecompose().Answer(ctx)`
- **Then** ответ генерируется с использованием декомпозиции запроса на под-вопросы
- Evidence: LLMProvider.Generate получает контекст, содержащий чанки по обоим аспектам; возвращённый ответ не пустой

### AC-002 LLM-based декомпозиция

- Почему это важно: LLM-декомпозиция даёт наиболее качественное разбиение сложных запросов.
- **Given** Pipeline с LLMProvider (Generate возвращает JSON-массив под-вопросов) и дефолтным decomposer
- **When** вызывается `Search("Что такое X и как его использовать с Y?").SubDecompose().Retrieve(ctx)`
- **Then** retrieval выполняется как минимум для 2 разных под-вопросов, результат содержит чанки по обоим темам
- Evidence: проверка, что количество sub-query retrieve вызовов >= 2 (через mock VectorStore)

### AC-003 Rule-based fallback

- Почему это важно: rule-based декомпозиция работает без LLM и обеспечивает graceful degradation.
- **Given** Pipeline без LLMProvider (nil) с rule-based decomposer
- **When** вызывается `Search("X и Y").SubDecompose().Retrieve(ctx)`
- **Then** запрос разбивается по разделителю "и", retrieval выполняется для "X" и "Y"
- Evidence: под-вопросы непустые и отличаются от исходного запроса

### AC-004 Merge результатов

- Почему это важно: один и тот же чанк может быть найден по нескольким под-вопросам; дедупликация обязательна.
- **Given** один и тот же релевантный чанк возвращается по двум разным под-вопросам
- **When** результаты retrieval объединяются
- **Then** чанк присутствует ровно один раз в merged результате, его score = максимальный среди всех под-вопросов
- Evidence: количество уникальных Chunk.ID в merged результате < суммарного raw количества

### AC-005 Graceful degradation при ошибке декомпозиции

- Почему это важно: ошибка decomposer'а не должна ломать запрос пользователя.
- **Given** Pipeline с LLM-декомпозицией, которая возвращает ошибку
- **When** вызывается `Search("X и Y").SubDecompose().Retrieve(ctx)`
- **Then** pipeline выполняет fallback на rule-based splitting; если и rule-based не дал под-вопросов — single-query retrieval с исходным query
- Evidence: при ошибке LLM retrieval выполняется с rule-based разбиением (вызовов VectorStore.Search >= 2 для запроса "X и Y")

### AC-006 Per-request override

- Почему это важно: пользователь может отключить декомпозицию для конкретного запроса, даже если она включена на уровне Pipeline.
- **Given** Pipeline с QueryDecomposer в опциях
- **When** вызывается `pipeline.Search("вопрос").Retrieve(ctx)` (без SubDecompose)
- **Then** используется обычный single-query retrieval, декомпозиция не выполняется
- Evidence: VectorStore.Search вызывается ровно 1 раз

### AC-007 Parallel retrieval

- Почему это важно: параллельное выполнение под-вопросов снижает latency.
- **Given** Pipeline с decomposer'ом, возвращающим 3 под-вопроса
- **When** вызывается `Search("...").SubDecompose().Retrieve(ctx)`
- **Then** retrieve по под-вопросам выполняется параллельно (не последовательно)
- Evidence: mock VectorStore.Search вызывается 3 раза за одно конкурентное окно (time overlap)

### AC-008 Answer с декомпозицией

- Почему это важно: конечный ответ учитывает все аспекты исходного запроса.
- **Given** Pipeline с LLMProvider и decomposer'ом
- **When** вызывается `Search("Каковы требования к A и стоимость B?").SubDecompose().Answer(ctx)`
- **Then** ответ содержит информацию как по A, так и по B
- Evidence: в выводе LLMProvider.Generate присутствуют фрагменты, связанные с обоими аспектами (проверка через mock)

### AC-009 Совместимость с существующими методами

- Почему это важно: SubDecompose работает с существующим API (Cite, InlineCite, Stream).
- **Given** Pipeline с decomposer'ом
- **When** вызывается `Search("...").SubDecompose().Cite(ctx)`
- **Then** возвращается корректный ответ и источники
- Evidence: Cite возвращает непустой RetrievalResult (sources)

## Допущения

- LLM decomposer получает запрос вида «Разбей запрос на независимые под-вопросы. Ответь JSON-массивом строк.» — prompt настраивается, но дефолтный prompt часть реализации.
- Rule-based decomposer разбивает по союзам «и», «или», «,» — для русского и английского языка.
- Количество под-вопросов по умолчанию не больше 5 (настраивается).
- topK для каждого под-вопроса = topK исходного запроса (не настраивается по-отдельности).
- Все под-вопросы равноправны — нет взвешивания или приоритизации.
- Merge использует max score и дедупликацию по Chunk.ID — без дополнительной реранжировки на merged наборе.

## Критерии успеха

- SC-001 Ответ через SubDecompose покрывает больше релевантных аспектов, чем обычный Answer (оценка через LLM-as-judge или manual review).

## Краевые случаи

- Пустой запрос: SubDecompose на пустом запросе возвращает ErrEmptyQuery (как и обычный Search).
- Один под-вопрос: если decomposer вернул 1 под-вопрос, поведение идентично обычному single-query (без overhead на merge).
- Nil decomposer: SubDecompose() на Pipeline без decomposer'а возвращает ErrNotSupported.
- Context cancelled: cancellation во время декомпозиции отменяет все под-запросы (через derived context).
- Zero результатов: если все под-вопросы вернули пустой retrieval, Answer получает пустой контекст (как в обычном режиме).

## Открытые вопросы

- Должен ли decomposer поддерживать weighted под-вопросы (один аспект важнее другого)?
- Нужен ли streaming-режим для SubDecompose.Stream? Пока в AC — только Cite/Answer.
- Стоит ли добавить возможность ограничить max под-вопросов через PipelineOptions? Пока дефолт 5.
- Какой формат ответа LLM ожидать от decomposer'а? JSON-массив строк — но нужна устойчивость к отклонениям от формата.
- Должен ли decomposer использовать отдельный LLMProvider (с дешёвой моделью) или тот же, что и для генерации?
