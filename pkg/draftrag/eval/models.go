package eval

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
}

// Metrics — агрегированные метрики eval-прогона.
type Metrics struct {
	TotalCases int
	HitAtK     float64
	MRR        float64
}

// Report — агрегированный отчёт по датасету.
type Report struct {
	Metrics Metrics
	Cases   []CaseResult
}
