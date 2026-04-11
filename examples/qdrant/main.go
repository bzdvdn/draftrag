// Пример RAG с Qdrant — пример использования draftRAG.
//
// Создаёт коллекцию в Qdrant (если не существует), индексирует документы
// и запускает интерактивный RAG-чат.
//
// Быстрый старт с Docker:
//
//	docker run -p 6333:6333 qdrant/qdrant
//	EMBEDDER_API_KEY=sk-... LLM_API_KEY=sk-... \
//	  go run ./examples/qdrant/
//
// Переменные окружения:
//
//	QDRANT_URL          — URL Qdrant сервера (по умолчанию: http://localhost:6333)
//	QDRANT_COLLECTION   — имя коллекции (по умолчанию: draftrag_example)
//	EMBEDDER_BASE_URL   — базовый URL embedder API (по умолчанию: https://api.openai.com)
//	EMBEDDER_API_KEY    — ключ API для embedder (обязательно)
//	EMBEDDER_MODEL      — модель embeddings (по умолчанию: text-embedding-ada-002)
//	LLM_BASE_URL        — базовый URL LLM API (по умолчанию: https://api.openai.com)
//	LLM_API_KEY         — ключ API для LLM (обязательно)
//	LLM_MODEL           — модель LLM (по умолчанию: gpt-4o-mini)
//	EMBEDDING_DIM       — размерность векторов (по умолчанию: 1536 для ada-002)
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

// Пример документов — описание туристических направлений.
var documents = []draftrag.Document{
	{
		ID:      "destination-bali",
		Content: "Бали — остров богов в Индонезии с богатой культурой и живописной природой. Лучшее время для посещения: апрель-октябрь (сухой сезон). Главные достопримечательности: рисовые террасы Тегалаланг, храм Тананах Лот на скале в океане, вулкан Батур с треккингом на рассвете, культурная столица Убуд с художественными галереями и традиционными танцами. Денпасар — столица и транспортный хаб острова.",
		Metadata: map[string]string{"region": "asia", "country": "indonesia"},
	},
	{
		ID:      "destination-iceland",
		Content: "Исландия — страна льда и огня с уникальными природными явлениями. Северное сияние наблюдается с сентября по март. Золотое кольцо включает гейзер Гейсир, водопад Гюдльфосс и разломы тектонических плит Тингвеллир. Голубая лагуна — геотермальный спа-курорт. Рейкьявик — самая северная столица мира. Лучший сезон для пешего туризма: июнь-август. Полярный день летом позволяет гулять ночью.",
		Metadata: map[string]string{"region": "europe", "country": "iceland"},
	},
	{
		ID:      "destination-japan",
		Content: "Япония сочетает древние традиции и современные технологии. Токио — крупнейший мегаполис мира с районами Сибуя, Синдзюку и Акихабара. Сезон сакуры в марте-апреле привлекает миллионы туристов. Киото — древняя столица с тысячами храмов и чайными садами. Гора Фудзи — символ страны. Японская кухня: суши, рамен, темпура. Транспорт: Shinkansen (скоростные поезда) связывают все крупные города.",
		Metadata: map[string]string{"region": "asia", "country": "japan"},
	},
	{
		ID:      "destination-peru",
		Content: "Перу — страна инков с мировым наследием UNESCO. Мачу-Пикчу — затерянный город инков на высоте 2430 метров, построенный в XV веке. Тропа инков — 4-дневный треккинг через горы и джунгли. Куско — историческая столица инкской империи. Озеро Титикака — самое высокогорное судоходное озеро мира. Амазонские джунгли занимают 60% территории страны. Лучшее время: май-сентябрь.",
		Metadata: map[string]string{"region": "americas", "country": "peru"},
	},
	{
		ID:      "destination-morocco",
		Content: "Марокко — ворота Африки с богатым культурным наследием. Медина Марракеша — лабиринт улочек, шумные рынки (суки) и площадь Джемаа-эль-Фна с уличными артистами. Голубой город Шефшауэн расположен в горах Рифа. Пустыня Сахара в районе Мерзуги — верблюжьи туры и ночёвка в берберских шатрах. Фес — древнейший медресе мира. Касабланка — экономическая столица. Кухня: тажин, кускус, пастила.",
		Metadata: map[string]string{"region": "africa", "country": "morocco"},
	},
	{
		ID:      "destination-norway",
		Content: "Норвегия — страна фьордов, викингов и северного сияния. Гейрангерфьорд и Нэрёйфьорд включены в список UNESCO. Берген — ворота в страну фьордов с набережной Брюгген. Тролльтунга — скала над озером на высоте 1100 м, популярный треккинг. Лофотенские острова с рыбацкими деревнями идеальны для фотографии. Осло — столица с музеями вигеланда и Кон-Тики. Полярный круг пересекает страну примерно посередине.",
		Metadata: map[string]string{"region": "europe", "country": "norway"},
	},
}

func main() {
	ctx := context.Background()

	qdrantURL := envOrDefault("QDRANT_URL", "http://localhost:6333")
	collection := envOrDefault("QDRANT_COLLECTION", "draftrag_example")
	embeddingDim := envIntOrDefault("EMBEDDING_DIM", 1536)

	opts := draftrag.QdrantOptions{
		URL:        qdrantURL,
		Collection: collection,
		Dimension:  embeddingDim,
	}

	fmt.Printf("Подключаемся к Qdrant: %s\n", qdrantURL)

	// Создаём коллекцию, если она ещё не существует.
	exists, err := draftrag.CollectionExists(ctx, opts)
	if err != nil {
		fatalf("ошибка проверки коллекции: %v\n", err)
	}
	if !exists {
		fmt.Printf("Создаём коллекцию %q (dim=%d)...\n", collection, embeddingDim)
		if err := draftrag.CreateCollection(ctx, opts); err != nil {
			fatalf("ошибка создания коллекции: %v\n", err)
		}
		fmt.Println("Коллекция создана.")
	} else {
		fmt.Printf("Коллекция %q уже существует.\n", collection)
	}

	store, err := draftrag.NewQdrantStore(opts)
	if err != nil {
		fatalf("ошибка создания store: %v\n", err)
	}

	embedder := draftrag.NewOpenAICompatibleEmbedder(draftrag.OpenAICompatibleEmbedderOptions{
		BaseURL: envOrDefault("EMBEDDER_BASE_URL", "https://api.openai.com"),
		APIKey:  mustEnv("EMBEDDER_API_KEY"),
		Model:   envOrDefault("EMBEDDER_MODEL", "text-embedding-ada-002"),
	})

	llm := draftrag.NewOpenAICompatibleLLM(draftrag.OpenAICompatibleLLMOptions{
		BaseURL: envOrDefault("LLM_BASE_URL", "https://api.openai.com"),
		APIKey:  mustEnv("LLM_API_KEY"),
		Model:   envOrDefault("LLM_MODEL", "gpt-4o-mini"),
	})

	pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		DefaultTopK: 3,
		Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
			ChunkSize: 400,
			Overlap:   50,
		}),
	})

	fmt.Printf("Индексируем %d документов...\n", len(documents))
	if err := pipeline.Index(ctx, documents); err != nil {
		// Если Qdrant недоступен — выводим понятное сообщение.
		if errors.Is(err, draftrag.ErrEmptyDocument) {
			fatalf("ошибка: пустой документ в наборе\n")
		}
		fatalf("ошибка индексации: %v\n", err)
	}
	fmt.Println("Индексация завершена.")
	fmt.Println("\nRAG-чат с Qdrant готов. Введите вопрос (Ctrl+C для выхода):")
	fmt.Println(strings.Repeat("─", 60))

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("\n> ")
		if !scanner.Scan() {
			break
		}
		question := strings.TrimSpace(scanner.Text())
		if question == "" {
			continue
		}

		answer, sources, err := pipeline.Search(question).TopK(3).Cite(ctx)
		if err != nil {
			if errors.Is(err, draftrag.ErrFiltersNotSupported) {
				fmt.Fprintln(os.Stderr, "фильтры не поддерживаются этим store")
				continue
			}
			fmt.Fprintf(os.Stderr, "ошибка: %v\n", err)
			continue
		}

		fmt.Printf("\n%s\n", answer)

		if len(sources.Chunks) > 0 {
			fmt.Println("\nИсточники:")
			for i, r := range sources.Chunks {
				fmt.Printf("  [%d] %s (score=%.3f)\n", i+1, r.Chunk.ParentID, r.Score)
			}
		}

		fmt.Println(strings.Repeat("─", 60))
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntOrDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			return n
		}
	}
	return def
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		fatalf("переменная окружения %s не задана\n", key)
	}
	return v
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
