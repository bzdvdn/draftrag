package draftrag

import (
	"sort"
	"strings"
	"testing"
)

func TestPGVectorMigrationAssets_PresentAndDeterministic(t *testing.T) {
	paths, err := listPGVectorMigrationAssets()
	if err != nil {
		t.Fatalf("list assets: %v", err)
	}
	if len(paths) == 0 {
		t.Fatalf("expected non-empty migration assets")
	}

	// Проверяем детерминированный порядок (lexicographic).
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	for i := range paths {
		if paths[i] != sorted[i] {
			t.Fatalf("assets are not sorted: got=%v", paths)
		}
	}

	// Проверяем наличие “якорных” файлов.
	want := []string{
		pgvectorMigrationExtension,
		pgvectorMigrationV1,
		pgvectorMigrationV2,
	}
	for _, w := range want {
		found := false
		for _, p := range paths {
			if p == w {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing migration asset: %s; got=%v", w, paths)
		}
	}
}

func TestPGVectorMigrationAssets_ContainAnchors(t *testing.T) {
	cases := []struct {
		path  string
		parts []string
	}{
		{
			path: pgvectorMigrationExtension,
			parts: []string{
				"CREATE EXTENSION",
				"vector",
			},
		},
		{
			path: pgvectorMigrationV1,
			parts: []string{
				"CREATE TABLE IF NOT EXISTS",
				"{{TABLE}}",
				"VECTOR({{DIM}})",
			},
		},
		{
			path: pgvectorMigrationV2,
			parts: []string{
				"ALTER TABLE {{TABLE}}",
				"CREATE INDEX IF NOT EXISTS {{PARENT_ID_INDEX}}",
				"CREATE INDEX IF NOT EXISTS {{PARENT_POS_INDEX}}",
			},
		},
	}

	for _, tc := range cases {
		text, err := readPGVectorMigrationAsset(tc.path)
		if err != nil {
			t.Fatalf("read %s: %v", tc.path, err)
		}
		for _, part := range tc.parts {
			if !strings.Contains(text, part) {
				t.Fatalf("expected %s to contain %q", tc.path, part)
			}
		}
	}
}
