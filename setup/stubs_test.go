package main

import (
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigStub(t *testing.T) {
	tests := []struct {
		pkg, facades string
	}{
		{"config", "goravel/app/facades"},
		{"settings", "example.com/shop/app/facades"},
	}
	for _, tt := range tests {
		t.Run(tt.facades, func(t *testing.T) {
			stub := configStub(tt.pkg, tt.facades)
			file, err := parser.ParseFile(token.NewFileSet(), "money.go", stub, parser.ImportsOnly)
			require.NoError(t, err)
			assert.Equal(t, tt.pkg, file.Name.Name)
			require.Len(t, file.Imports, 1)
			assert.Equal(t, `"`+tt.facades+`"`, file.Imports[0].Path.Value)
			assert.Contains(t, stub, `"default_currency": config.Env("MONEY_CURRENCY", "USD")`)
			for _, key := range []string{`"rounding": "half_up"`, `"currencies": map[string]any{`, `"locale": ""`, `"amount":            "minor"`, `"include_decimal":   true`, `"include_formatted": true`} {
				assert.Contains(t, stub, key)
			}
			_, err = parser.ParseFile(token.NewFileSet(), "money.go", stub, 0)
			require.NoError(t, err, "the stub is valid Go")
			assert.NotContains(t, stub, "Dummy")
		})
	}
}
