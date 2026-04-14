package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseKeywords_CSVAware(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty", "", []string{}},
		{"spaces only", "   ", []string{}},
		{"simple comma", "a, b, c", []string{"a", "b", "c"}},
		{"quoted phrase no comma", `aposentadoria, "preciso revisar"`, []string{"aposentadoria", "preciso revisar"}},
		{"quoted phrase with comma", `a, "frase, com virgula", b`, []string{"a", "frase, com virgula", "b"}},
		{"multiple quoted", `"a, b", "c, d"`, []string{"a, b", "c, d"}},
		{"trailing spaces", `a,  "x, y"  , b`, []string{"a", "x, y", "b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseKeywords(tc.input)
			if len(tc.want) == 0 && len(got) == 0 {
				return
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
