package textutil_test

import (
	"testing"

	"github.com/reinoplus/reinoplus/internal/textutil"
)

func TestTitleCaseName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "PEDRO ICARO COSTA NEVES", want: "Pedro Icaro Costa Neves"},
		{input: "JOAO DA SILVA", want: "Joao da Silva"},
		{input: "  maria da silva  ", want: "Maria da Silva"},
		{input: "ANA DE SOUSA", want: "Ana de Sousa"},
		{input: "PEDRO DOS SANTOS", want: "Pedro dos Santos"},
		{input: "MARIA DAS DORES", want: "Maria das Dores"},
		{input: "josé", want: "José"},
		{input: "", want: ""},
		{input: "Pedro Icaro Costa Neves", want: "Pedro Icaro Costa Neves"},
	}

	for _, tt := range tests {
		if got := textutil.TitleCaseName(tt.input); got != tt.want {
			t.Fatalf("TitleCaseName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTitleCaseNamePtr(t *testing.T) {
	value := "JOAO DA SILVA"
	got := textutil.TitleCaseNamePtr(&value)
	if got == nil || *got != "Joao da Silva" {
		t.Fatalf("TitleCaseNamePtr() = %v, want Joao da Silva", got)
	}

	if textutil.TitleCaseNamePtr(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}
