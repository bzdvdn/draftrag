// Базовый RAG-чат (CLI) — пример использования draftRAG.
//
// Загружает набор документов в in-memory store, затем запускает
// интерактивный цикл вопрос-ответ с inline-цитатами.
//
// Переменные окружения:
//
//	EMBEDDER_BASE_URL   — базовый URL embedder API (по умолчанию: https://api.openai.com)
//	EMBEDDER_API_KEY    — ключ API для embedder (обязательно)
//	EMBEDDER_MODEL      — модель embeddings (по умолчанию: text-embedding-ada-002)
//	LLM_BASE_URL        — базовый URL LLM API (по умолчанию: https://api.openai.com)
//	LLM_API_KEY         — ключ API для LLM (обязательно)
//	LLM_MODEL           — модель LLM (по умолчанию: gpt-4o-mini)
//
// Ollama (локальный режим):
//
//	Установите EMBEDDER_BASE_URL=http://localhost:11434 и EMBEDDER_MODEL=nomic-embed-text,
//	LLM_BASE_URL=http://localhost:11434 и LLM_MODEL=llama3.2.
//
// Запуск:
//
//	EMBEDDER_API_KEY=sk-... LLM_API_KEY=sk-... go run ./examples/chat/
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bzdvdn/draftrag/pkg/draftrag"
)

// Знания о продукте — SmartHome Hub (пример корпоративной базы знаний).
var knowledgeBase = []draftrag.Document{
	{
		ID:      "smarthome-overview",
		Content: "SmartHome Hub — центральный контроллер умного дома, который объединяет устройства разных производителей через единый интерфейс. Поддерживает Zigbee, Z-Wave, Wi-Fi и Matter протоколы. Управление доступно через мобильное приложение SmartHome для iOS и Android, а также через веб-панель. Локальная обработка данных обеспечивает работу без интернета.",
		Metadata: map[string]string{"category": "overview", "product": "smarthome-hub"},
	},
	{
		ID:      "smarthome-setup",
		Content: "Первичная настройка SmartHome Hub занимает 5–10 минут. Подключите устройство к роутеру через Ethernet или Wi-Fi, установите приложение SmartHome, создайте аккаунт и отсканируйте QR-код на нижней панели устройства. После активации хаб автоматически обнаружит совместимые устройства в зоне действия. Рекомендуется размещать хаб в центре квартиры для максимального покрытия.",
		Metadata: map[string]string{"category": "setup", "product": "smarthome-hub"},
	},
	{
		ID:      "smarthome-zigbee",
		Content: "Для добавления Zigbee-устройства откройте приложение, выберите «Добавить устройство» → «Zigbee». Переведите устройство в режим сопряжения (обычно зажав кнопку на 5 секунд до мигания). Хаб обнаружит устройство в течение 30 секунд. SmartHome Hub поддерживает до 200 Zigbee-устройств одновременно. Устройства также могут работать в режиме роутера, расширяя зону покрытия сети.",
		Metadata: map[string]string{"category": "zigbee", "product": "smarthome-hub"},
	},
	{
		ID:      "smarthome-automations",
		Content: "Автоматизации в SmartHome Hub создаются через раздел «Сцены и автоматизации». Поддерживаются триггеры: время суток, восход/закат, состояние устройства, геолокация (приход/уход), показатели датчиков (температура, влажность, освещённость). Действия могут включать управление освройствами, отправку push-уведомлений, запуск сцен. Автоматизации выполняются локально без интернета.",
		Metadata: map[string]string{"category": "automations", "product": "smarthome-hub"},
	},
	{
		ID:      "smarthome-security",
		Content: "Модуль безопасности SmartHome Hub поддерживает датчики открытия дверей и окон, датчики движения PIR, сирены и умные замки. При срабатывании охраны можно настроить: запись с камер, включение сирены, уведомления в приложение, звонок на номер экстренной связи. Режимы охраны: «Дома», «Ушёл», «Ночь» с разными наборами активных датчиков.",
		Metadata: map[string]string{"category": "security", "product": "smarthome-hub"},
	},
	{
		ID:      "smarthome-energy",
		Content: "Мониторинг энергопотребления доступен для устройств с поддержкой измерения мощности. В приложении отображаются графики потребления по устройствам, комнатам и в целом по дому. SmartHome Hub умеет автоматически выключать устройства при превышении заданного порога потребления или по расписанию. Интеграция с тарифами ТЭК позволяет рассчитывать стоимость потребления.",
		Metadata: map[string]string{"category": "energy", "product": "smarthome-hub"},
	},
	{
		ID:      "smarthome-voice",
		Content: "SmartHome Hub интегрируется с голосовыми ассистентами: Алиса (Яндекс), Google Home и Amazon Alexa. Для подключения Алисы перейдите в настройки приложения → «Голосовые ассистенты» → «Яндекс Алиса» и авторизуйтесь. После этого все устройства появятся в приложении Яндекс. Поддерживаются команды включения/выключения, управления яркостью, изменения температуры термостата.",
		Metadata: map[string]string{"category": "voice", "product": "smarthome-hub"},
	},
	{
		ID:      "smarthome-troubleshooting",
		Content: "Частые проблемы и решения. Устройство недоступно: проверьте питание и расстояние до хаба, переподключите устройство. Приложение не видит хаб: убедитесь что хаб и телефон в одной сети, перезапустите приложение. Долгая реакция устройств: хаб перегружен, ограничьте количество одновременных автоматизаций. Сброс к заводским настройкам: удерживайте кнопку Reset 10 секунд до красного мигания.",
		Metadata: map[string]string{"category": "troubleshooting", "product": "smarthome-hub"},
	},
}

func main() {
	ctx := context.Background()

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

	store := draftrag.NewInMemoryStore()
	pipeline := draftrag.NewPipelineWithOptions(store, llm, embedder, draftrag.PipelineOptions{
		DefaultTopK: 3,
		Chunker: draftrag.NewBasicChunker(draftrag.BasicChunkerOptions{
			ChunkSize: 300,
			Overlap:   40,
		}),
	})

	fmt.Println("Индексируем базу знаний...")
	if err := pipeline.Index(ctx, knowledgeBase); err != nil {
		fatalf("ошибка индексации: %v\n", err)
	}
	fmt.Printf("Проиндексировано %d документов.\n\n", len(knowledgeBase))

	fmt.Println("RAG-чат готов. Введите вопрос (Ctrl+C для выхода):")
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

		answer, _, citations, err := pipeline.Search(question).InlineCite(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ошибка: %v\n", err)
			continue
		}

		fmt.Printf("\n%s\n", answer)

		if len(citations) > 0 {
			fmt.Println("\nИсточники:")
			seen := map[string]bool{}
			for _, c := range citations {
				id := c.Chunk.Chunk.ParentID
				if seen[id] {
					continue
				}
				seen[id] = true
				fmt.Printf("  [%d] %s (score=%.3f)\n", c.Number, id, c.Chunk.Score)
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
