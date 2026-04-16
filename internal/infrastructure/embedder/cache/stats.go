package cache

import (
	"sync/atomic"
)

// Stats содержит метрики работы кэша.
// @ds-task T1.3: Структура для сбора статистики кэша (AC-007)
type Stats struct {
	Hits      uint64 // количество попаданий в кэш
	Misses    uint64 // количество промахов
	Evictions uint64 // количество вытесненных записей
}

// HitRate возвращает долю попаданий в кэш (от 0 до 1).
// Возвращает 0, если не было ни одного обращения.
// @ds-task T1.3: Метод вычисления hit rate (AC-007)
func (s Stats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// statsCollector собирает метрики кэша через atomic операции.
type statsCollector struct {
	hits      atomic.Uint64
	misses    atomic.Uint64
	evictions atomic.Uint64
}

// RecordHit инкрементирует счётчик попаданий.
// @ds-task T1.3: Метод записи hit (AC-007)
func (s *statsCollector) RecordHit() {
	s.hits.Add(1)
}

// RecordMiss инкрементирует счётчик промахов.
// @ds-task T1.3: Метод записи miss (AC-007)
func (s *statsCollector) RecordMiss() {
	s.misses.Add(1)
}

// RecordEviction инкрементирует счётчик вытеснений.
// @ds-task T1.3: Метод записи eviction (AC-007)
func (s *statsCollector) RecordEviction() {
	s.evictions.Add(1)
}

// Stats возвращает текущие метрики кэша.
// @ds-task T1.3: Метод получения статистики (AC-007)
func (s *statsCollector) Stats() Stats {
	return Stats{
		Hits:      s.hits.Load(),
		Misses:    s.misses.Load(),
		Evictions: s.evictions.Load(),
	}
}
