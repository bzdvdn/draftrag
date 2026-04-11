package domain

import "testing"

func TestDocumentValidate_EmptyContent(t *testing.T) {
	doc := Document{
		ID:      "doc-1",
		Content: "",
	}

	if err := doc.Validate(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestChunkValidate_EmptyParentID(t *testing.T) {
	chunk := Chunk{
		ID:       "chunk-1",
		Content:  "hello",
		ParentID: "",
		Position: 0,
	}

	if err := chunk.Validate(); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestDefaultHybridConfig(t *testing.T) {
	cfg := DefaultHybridConfig()

	if cfg.SemanticWeight != 0.7 {
		t.Errorf("expected SemanticWeight=0.7, got %f", cfg.SemanticWeight)
	}
	if !cfg.UseRRF {
		t.Error("expected UseRRF=true")
	}
	if cfg.RRFK != 60 {
		t.Errorf("expected RRFK=60, got %d", cfg.RRFK)
	}
	if cfg.BMFinalK != 0 {
		t.Errorf("expected BMFinalK=0, got %d", cfg.BMFinalK)
	}

	// Дефолтная конфигурация должна проходить валидацию
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid: %v", err)
	}
}

func TestHybridConfig_Validate_SemanticWeight(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
		valid  bool
	}{
		{"negative", -0.1, false},
		{"zero", 0.0, true},
		{"one", 1.0, true},
		{"valid", 0.7, true},
		{"greater than one", 1.1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := HybridConfig{SemanticWeight: tt.weight, RRFK: 60, BMFinalK: 0}
			err := cfg.Validate()
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestHybridConfig_Validate_RRFK(t *testing.T) {
	tests := []struct {
		name  string
		rrfk  int
		valid bool
	}{
		{"zero", 0, false},
		{"negative", -1, false},
		{"one", 1, true},
		{"default", 60, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := HybridConfig{SemanticWeight: 0.7, RRFK: tt.rrfk, BMFinalK: 0}
			err := cfg.Validate()
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestHybridConfig_Validate_BMFinalK(t *testing.T) {
	tests := []struct {
		name     string
		bmFinaKK int
		valid    bool
	}{
		{"negative", -1, false},
		{"zero", 0, true},
		{"positive", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := HybridConfig{SemanticWeight: 0.7, RRFK: 60, BMFinalK: tt.bmFinaKK}
			err := cfg.Validate()
			if tt.valid && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
