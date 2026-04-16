package eval

import (
	"encoding/json"
)

// Case описывает один eval-кейс для оценки retrieval.
type Case struct {
	// ID — стабильный идентификатор кейса (опционально).
	ID string
	// Question — текст вопроса, который будет отправлен в retrieval.
	Question string
	// ExpectedParentIDs — множество “правильных” parent IDs, по которым оценивается hit@k и rank.
	ExpectedParentIDs []string
	// TopK — topK для retrieval в рамках кейса; 0 означает использование дефолта harness.
	TopK int
}

// CaseResult — подробный результат прогона одного кейса.
type CaseResult struct {
	CaseID string

	// Found — попадание хотя бы одного ExpectedParentIDs в topK.
	Found bool
	// Rank — 1..K для первого попадания; 0 если не найдено.
	Rank int

	// RetrievedParentIDs — parentIDs в порядке ранжирования (для дебага).
	RetrievedParentIDs []string

	// @sk-task T1.2: Добавить поле NDCG для per-case оценки ранжирования (AC-004)
	NDCG float64
	// @sk-task T1.2: Добавить поле Precision для per-case оценки точности (AC-004)
	Precision float64
	// @sk-task T1.2: Добавить поле Recall для per-case оценки полноты (AC-004)
	Recall float64
}

// Metrics — агрегированные метрики eval-прогона.
type Metrics struct {
	TotalCases int
	HitAtK     float64
	MRR        float64
	// @sk-task T1.1: Добавить поле NDCG для оценки ранжирования с учётом релевантности (AC-001)
	NDCG float64
	// @sk-task T1.1: Добавить поле Precision для оценки точности поиска (AC-002)
	Precision float64
	// @sk-task T1.1: Добавить поле Recall для оценки полноты поиска (AC-002)
	Recall float64
}

// Report — агрегированный отчёт по датасету.
type Report struct {
	Metrics Metrics
	Cases   []CaseResult
}

// MarshalJSON реализует сериализацию Report в JSON.
// @sk-task T2.4: MarshalJSON реализует сериализацию Report в JSON (AC-006)
func (r Report) MarshalJSON() ([]byte, error) {
	type Alias Report
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(&r),
	})
}
