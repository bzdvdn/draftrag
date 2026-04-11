package draftrag

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// pgvectorMigrationAssetsFS содержит SQL-миграции pgvector store.
//
//go:embed migrations/pgvector/*.sql
var pgvectorMigrationAssetsFS embed.FS

func listPGVectorMigrationAssets() ([]string, error) {
	paths, err := fs.Glob(pgvectorMigrationAssetsFS, "migrations/pgvector/*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

func readPGVectorMigrationAsset(path string) (string, error) {
	data, err := pgvectorMigrationAssetsFS.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func renderPGVectorSQLTemplate(tpl string, replacements map[string]string) (string, error) {
	if len(replacements) == 0 {
		return tpl, nil
	}

	pairs := make([]string, 0, len(replacements)*2)
	for k, v := range replacements {
		pairs = append(pairs, "{{"+k+"}}", v)
	}
	out := strings.NewReplacer(pairs...).Replace(tpl)
	if strings.Contains(out, "{{") {
		return "", fmt.Errorf("unresolved template placeholders in migration SQL")
	}
	return out, nil
}
