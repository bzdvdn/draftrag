# ROADMAP — draftRAG

Дата последнего обновления: 2026-04-09

---

## Реализовано ✅

| Фича | Статус | Примечание |
|---|---|---|
| Core interfaces (VectorStore, LLMProvider, Embedder, Chunker) | ✅ | Clean Architecture |
| In-memory vector store | ✅ | Для тестов и прототипов |
| pgvector (PostgreSQL) vector store | ✅ | Полная поддержка: фильтры, миграции, гибридный поиск |
| Qdrant vector store | ✅ | REST API, payload-фильтры, управление коллекциями |
| ChromaDB vector store | ✅ | Базовая поддержка (без гибридного поиска) |
| OpenAI-compatible embedder | ✅ | Все модели embeddings |
| Ollama embedder | ✅ | Локальные embedding-модели |
| Cached embedder (LRU) | ✅ | Кэширование по хэшу текста |
| OpenAI-compatible LLM | ✅ | Responses API, streaming |
| Anthropic Claude LLM | ✅ | Нативный Messages API, streaming |
| Ollama LLM | ✅ | Локальные LLM |
| Basic rune chunker | ✅ | С настраиваемым overlap |
| Pipeline: index, query, answer | ✅ | IndexBatch с обработкой ошибок |
| Answer с citations и inline citations `[1]` | ✅ | Полная поддержка |
| Streaming ответов | ✅ | `AnswerStream`, `AnswerStreamWithInlineCitations` |
| Metadata filtering | ✅ | `MetadataFilter` во всех хранилищах |
| Hybrid search (BM25 + semantic) | ✅ | Только pgvector (через `tsvector`) |
| HyDE (Hypothetical Document Embeddings) | ✅ | `Search().HyDE().Answer()` |
| Multi-query retrieval | ✅ | Несколько перефраз с объединением |
| Deduplication по ParentID | ✅ | Автоматическая |
| MMR reranking | ✅ | Диверсификация контекста |
| Retry + Circuit Breaker | ✅ | `RetryEmbedder`, `RetryLLMProvider` |
| Observability hooks | ✅ | Метрики всех стадий |
| Eval harness (Hit@K, MRR) | ✅ | Базовые метрики retrieval |
| pgvector migrations | ✅ | Версионированные миграции |

---

## Приоритет 1 — Следующий этап

### ChromaDB: управление коллекциями ✅
ChromaDB теперь поддерживает полный набор функций управления коллекциями:
- ✅ Миграции: `CreateCollection`, `DeleteCollection`, `CollectionExists`
- ✅ Консистентный API с другими хранилищами (переименование функций)

**Ограничение:** Гибридный поиск (BM25) **не поддерживается** — ChromaDB не имеет нативной реализации BM25. Используйте pgvector или Qdrant для гибридного поиска.

---

### Eval harness: расширенные метрики
Текущий eval harness измеряет только retrieval (Hit@K, MRR). Для production нужны метрики качества генерации.

**Что нужно:**
- Faithfulness: соответствует ли ответ источникам
- Context relevance: насколько извлечённый контекст релевантен вопросу
- Answer relevance: насколько ответ релевантен вопросу (RAGAS-style)
- A/B сравнение конфигураций pipeline

---

### Документация и примеры ✅
Структурированная документация и рабочие примеры реализованы.

**Выполнено:**
- `docs/` с полным покрытием: getting-started, concepts, pipeline-api, stores, embedders, llm, chunking, advanced, production, compatibility
- `examples/` с рабочим кодом: chat-cli, index-directory, pgvector-docker, qdrant-quickstart

---

## Приоритет 2 — Среднесрочно

### Query rewriting
Переформулировка вопроса через LLM перед поиском. HyDE уже реализован, но нет явного query rewriting.

**Что нужно:**
- `Search().Rewrite(prompt).Answer()` — явное переформулирование
- Интеграция с Multi-query: переписанный запрос → несколько вариантов

---

### Additional vector stores

**Weaviate** ✅ — Production-ready; basic retrieval, фильтры, управление коллекциями; **hybrid search не поддерживается**
**Milvus/Zilliz** — высокопроизводительный distributed векторный поиск
**Pinecone** — managed vector DB (требует API key, ограничения бесплатного tier)

---

### Embedder cache: Redis backend
Текущий `CachedEmbedder` использует только in-memory LRU. Для горизонтального масштабирования нужен Redis.

**Что нужно:**
- Интерфейс `CacheBackend` (in-memory / Redis)
- Redis implementation с TTL
- Сериализация векторов (msgpack)

---

## Приоритет 3 — Долгосрочно / Исследования

### Advanced RAG techniques

| Техника | Описание | Статус |
|---|---|---|
| Re-ranking (cross-encoder) | Переранжирование через Cohere Rerank или локальный cross-encoder | Интерфейс есть, реализаций нет |
| Contextual chunking | Чанкинг с учётом контекста документа (не просто по размеру) | Исследовать |
| Hierarchical indices | Два уровня: parent document + chunks | Исследовать |
| Sub-query decomposition | Разбиение сложных вопросов на под-вопросы | Исследовать |

---

### Production ops

- **Health checks**: endpoint для проверки состояния хранилищ и LLM
- **Metrics export**: Prometheus/OpenTelemetry интеграция
- **Graceful degradation**: fallback на меньшие модели при недоступности основных
- **Cost tracking**: подсчёт токенов/запросов для LLM API

---

### Многоязычность

- Токенизация для non-Unicode языков
- Поддержка CJK в чанкере
- Локализация prompt'ов

---

## Рекомендуемый порядок реализации

```
ChromaDB миграции
    → Eval harness: faithfulness + context relevance
        → Документация (docs/ + examples/)
            → Query rewriting
                → Redis cache backend
                    → Дополнительные vector stores (Weaviate/Milvus)
                        → Advanced RAG (cross-encoder reranking)
```

---

## Легенда

- ✅ Реализовано и протестировано
- 🚧 В разработке / частично
- 📋 Запланировано
- ❌ Не планируется (или требует обсуждения)
