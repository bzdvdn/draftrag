package draftrag

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestPublicAPI_OptionsPattern_Guardrail(t *testing.T) {
	t.Helper()

	// Явные исключения из правила. Должны быть legacy-формой и помечены Deprecated.
	allowMoreThanOneOptionsParam := map[string]string{
		"NewPGVectorStoreWithRuntimeOptions": "legacy API: сохранён для backward compatibility; используйте NewPGVectorStoreWithOptions",
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(
		fset,
		".",
		func(fi os.FileInfo) bool {
			// Этот тест живёт в pkg/draftrag; парсим только исходники пакета и исключаем *_test.go.
			name := fi.Name()
			return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
		},
		parser.ParseComments,
	)
	if err != nil {
		t.Fatalf("parse dir: %v", err)
	}

	pkg, ok := pkgs["draftrag"]
	if !ok {
		t.Fatalf("package draftrag not found")
	}

	// Собираем set типов *Options, которые действительно struct в этом пакете.
	structOptions := map[string]struct{}{}
	for _, f := range pkg.Files {
		for _, decl := range f.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if !strings.HasSuffix(ts.Name.Name, "Options") {
					continue
				}
				if _, ok := ts.Type.(*ast.StructType); ok {
					structOptions[ts.Name.Name] = struct{}{}
				}
			}
		}
	}

	var violations []string
	for _, f := range pkg.Files {
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Recv != nil || fd.Name == nil {
				continue
			}
			if !fd.Name.IsExported() || !strings.HasPrefix(fd.Name.Name, "New") {
				continue
			}

			optionsFields := 0
			optionsFieldIsLast := false
			if fd.Type.Params != nil && len(fd.Type.Params.List) > 0 {
				for idx, field := range fd.Type.Params.List {
					optionsTypeName, isOptions := extractLocalOptionsTypeName(field.Type, structOptions)
					if !isOptions {
						continue
					}
					_ = optionsTypeName
					optionsFields++
					if idx == len(fd.Type.Params.List)-1 {
						optionsFieldIsLast = true
					}
				}
			}

			if optionsFields == 0 {
				continue
			}

			if optionsFields == 1 && !optionsFieldIsLast {
				violations = append(violations, fd.Name.Name+": options param must be the last parameter")
				continue
			}

			if optionsFields > 1 {
				if _, allowed := allowMoreThanOneOptionsParam[fd.Name.Name]; !allowed {
					violations = append(violations, fd.Name.Name+": has more than one ...Options struct parameter (not allowlisted)")
					continue
				}
				if !hasDeprecatedDoc(fd.Doc) {
					violations = append(violations, fd.Name.Name+": allowlisted legacy API must have Deprecated doc comment")
					continue
				}
			}
		}
	}

	if len(violations) > 0 {
		t.Fatalf("public API options pattern violations:\n- %s", strings.Join(violations, "\n- "))
	}
}

func extractLocalOptionsTypeName(expr ast.Expr, localStructOptions map[string]struct{}) (string, bool) {
	switch e := expr.(type) {
	case *ast.Ident:
		_, ok := localStructOptions[e.Name]
		return e.Name, ok
	case *ast.StarExpr:
		return extractLocalOptionsTypeName(e.X, localStructOptions)
	default:
		return "", false
	}
}

func hasDeprecatedDoc(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	return strings.Contains(doc.Text(), "Deprecated:")
}
