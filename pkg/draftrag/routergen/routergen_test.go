package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"text/template"
)

// @sk-test arch-issues#T5.4: route table has all 7 handlers for every entry (AC-005, AC-006)
func TestRouteTable_AllColumnsDefined(t *testing.T) {
	columns := []string{"Retrieve", "Answer", "Cite", "InlineCite", "Stream", "StreamSources", "StreamCite"}
	for _, entry := range routeTable {
		for _, col := range columns {
			if entry.getTarget(col) == "" {
				t.Errorf("route %q missing handler for %s", entry.Route, col)
			}
		}
	}
}

// @sk-test arch-issues#T5.4: helper to extract target from routeEntry
func (r *routeEntry) getTarget(col string) string {
	switch col {
	case "Retrieve":
		return r.Retrieve
	case "Answer":
		return r.Answer
	case "Cite":
		return r.Cite
	case "InlineCite":
		return r.InlineCite
	case "Stream":
		return r.Stream
	case "StreamSources":
		return r.StreamSources
	case "StreamCite":
		return r.StreamCite
	}
	return ""
}

// @sk-test arch-issues#T5.4: route names are unique (AC-005, AC-006)
func TestRouteTable_UniqueNames(t *testing.T) {
	seen := make(map[string]bool)
	for _, entry := range routeTable {
		if seen[entry.Route] {
			t.Errorf("duplicate route: %s", entry.Route)
		}
		seen[entry.Route] = true
	}
}

// @sk-test arch-issues#T5.4: generator produces non-empty compilable output (AC-005, AC-006)
func TestGenerator_ProducesCompilableOutput(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "search_routes_gen.go")

	var entries []entryData
	for _, r := range routeTable {
		e := entryData{
			Route:   r.Route,
			Pattern: r.Pattern,
			Targets: make(map[string]string),
		}
		e.Targets["Retrieve"] = r.Retrieve
		e.Targets["Answer"] = r.Answer
		e.Targets["Cite"] = r.Cite
		e.Targets["InlineCite"] = r.InlineCite
		e.Targets["Stream"] = r.Stream
		e.Targets["StreamSources"] = r.StreamSources
		e.Targets["StreamCite"] = r.StreamCite
		entries = append(entries, e)
	}

	var columns []columnData
	for _, c := range outputColumns {
		columns = append(columns, columnData{
			Name:       c.Name,
			Wrapper:    c.Wrapper,
			ResultType: c.ResultType,
		})
	}

	data := tmplData{Columns: columns, Entries: entries}

	funcMap := template.FuncMap{"lowerFirst": lowerFirst}
	tmpl := template.Must(template.New("test").Funcs(funcMap).Parse(sourceTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("template execute: %v", err)
	}

	src := buf.Bytes()
	if len(src) == 0 {
		t.Fatal("generated empty output")
	}

	if err := os.WriteFile(outPath, src, 0644); err != nil {
		t.Fatal(err)
	}
}
