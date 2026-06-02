package application

import (
	"strconv"
	"strings"

	"github.com/bzdvdn/draftrag/internal/domain"
)

// @sk-task hardening-2026q2#T1.1: Разделить pipeline.go на модули (AC-001, AC-003)
func buildUserMessageV1(result domain.RetrievalResult, question string, maxContextChars, maxContextChunks int) string {
	contextText := buildContextTextV1(result, maxContextChars, maxContextChunks)

	var b strings.Builder
	b.WriteString("Контекст:\n")
	b.WriteString(contextText)
	// Гарантируем одну пустую строку между контекстом и вопросом (как в Prompt Contract v1),
	// независимо от того, был ли контекст обрезан по символам.
	if contextText == "" || strings.HasSuffix(contextText, "\n") {
		b.WriteString("\nВопрос:\n")
	} else {
		b.WriteString("\n\nВопрос:\n")
	}
	b.WriteString(question)
	return b.String()
}

func buildUserMessageV1InlineCitations(
	result domain.RetrievalResult,
	question string,
	maxContextChars, maxContextChunks int,
) (string, []domain.InlineCitation) {
	contextText, citations := buildContextTextV1InlineCitations(result, maxContextChars, maxContextChunks)

	var b strings.Builder

	b.WriteString("Инструкция:\n")
	b.WriteString("- В тексте ответа добавляй ссылки на источники в формате [n].\n")
	b.WriteString("- Используй только номера, которые есть в списке источников.\n\n")

	b.WriteString("Источники:\n")
	b.WriteString(contextText)
	// Гарантируем одну пустую строку между источниками и вопросом.
	if contextText == "" || strings.HasSuffix(contextText, "\n") {
		b.WriteString("\nВопрос:\n")
	} else {
		b.WriteString("\n\nВопрос:\n")
	}
	b.WriteString(question)

	return b.String(), citations
}

func buildContextTextV1(result domain.RetrievalResult, maxContextChars, maxContextChunks int) string {
	var b strings.Builder

	wroteChunks := 0
	for _, rc := range result.Chunks {
		if maxContextChunks > 0 && wroteChunks >= maxContextChunks {
			break
		}
		b.WriteString(rc.Chunk.Content)
		b.WriteString("\n")
		wroteChunks++
	}

	context := b.String()
	if maxContextChars <= 0 {
		return context
	}

	runes := []rune(context)
	if len(runes) <= maxContextChars {
		return context
	}
	return string(runes[:maxContextChars])
}

func buildContextTextV1InlineCitations(
	result domain.RetrievalResult,
	maxContextChars, maxContextChunks int,
) (string, []domain.InlineCitation) {
	var b strings.Builder
	citations := make([]domain.InlineCitation, 0, len(result.Chunks))

	runesWritten := 0
	wroteChunks := 0

	for _, rc := range result.Chunks {
		if maxContextChunks > 0 && wroteChunks >= maxContextChunks {
			break
		}

		number := wroteChunks + 1
		marker := "[" + strconv.Itoa(number) + "]"
		line := marker + " " + rc.Chunk.Content + "\n"

		if maxContextChars > 0 {
			lineRunes := []rune(line)
			if runesWritten+len(lineRunes) > maxContextChars {
				remaining := maxContextChars - runesWritten
				// Если не влезает даже маркер — ничего не добавляем и завершаем.
				if remaining <= len([]rune(marker)) {
					break
				}
				b.WriteString(string(lineRunes[:remaining]))
				citations = append(citations, domain.InlineCitation{
					Number: number,
					Chunk:  rc,
				})
				break
			}
		}

		b.WriteString(line)
		if maxContextChars > 0 {
			runesWritten += len([]rune(line))
		}
		citations = append(citations, domain.InlineCitation{
			Number: number,
			Chunk:  rc,
		})
		wroteChunks++
	}

	return b.String(), citations
}
