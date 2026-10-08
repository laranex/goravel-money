package money

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// declaredNames returns every top-level and method name declared in the
// package's non-test Go files.
func declaredNames(t *testing.T) map[string]bool {
	t.Helper()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err)
	names := map[string]bool{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		require.NoError(t, err)
		for _, decl := range parsed.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				names[d.Name.Name] = true
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						names[s.Name.Name] = true
						if st, ok := s.Type.(*ast.StructType); ok {
							for _, field := range st.Fields.List {
								for _, name := range field.Names {
									names[name.Name] = true
								}
							}
						}
					case *ast.ValueSpec:
						for _, name := range s.Names {
							names[name.Name] = true
						}
					}
				}
			}
		}
	}

	return names
}

// TestSkillOnlyUsesExistingAPI checks that every money.X identifier and every
// method the agent skill calls exists, so the skill cannot drift from the code.
func TestSkillOnlyUsesExistingAPI(t *testing.T) {
	skill, err := os.ReadFile("skills/goravel-money/SKILL.md")
	require.NoError(t, err)
	text := string(skill)
	names := declaredNames(t)

	for _, match := range regexp.MustCompile(`\bmoney\.([A-Z]\w*)`).FindAllStringSubmatch(text, -1) {
		assert.True(t, names[match[1]], "money.%s is not declared", match[1])
	}

	// Goravel, stdlib and Go-syntax calls that appear in the examples.
	external := map[string]bool{
		"Money": true, "Request": true, "Input": true, "Response": true, "Json": true, "Orm": true,
		"Query": true, "Create": true, "BigInteger": true, "Decimal": true, "Total": true, "Places": true,
		"Nullable": true, "String": true, "Env": true, "Is": true, "Marshal": true, "Code": true, "Currency": true,
	}
	for _, match := range regexp.MustCompile(`\.([A-Z]\w*)\(`).FindAllStringSubmatch(text, -1) {
		assert.True(t, names[match[1]] || external[match[1]], ".%s( is not part of the package", match[1])
	}

	assert.True(t, strings.HasPrefix(text, "---\nname: goravel-money\n"))
	for _, heading := range []string{"## When to use", "## Install", "## Configure", "## Use", "## Test your app", "## Avoid"} {
		assert.Contains(t, text, heading)
	}
}
