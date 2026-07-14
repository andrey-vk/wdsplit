package v2fly

import (
	"strings"
	"testing"

	"github.com/andrey-vk/wdsplit/internal/adapter"
	"github.com/andrey-vk/wdsplit/internal/model"
)

func TestParse(t *testing.T) {
	input := `# a comment line
full:exact.example.com
domain:suffix.example.com
keyword:midstring
bare-domain.example.com

  indented.example.com
example.com @cn
example.com @cn @ads
inline.example.com # trailing comment
regexp:^example\.com$
include:other-file
`

	entries, err := Parser{}.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	want := []adapter.Entry{
		{Type: model.EntryDomain, Value: "exact.example.com"},
		{Type: model.EntryDomainSuffix, Value: "suffix.example.com"},
		{Type: model.EntryDomainKeyword, Value: "midstring"},
		{Type: model.EntryDomainSuffix, Value: "bare-domain.example.com"},
		{Type: model.EntryDomainSuffix, Value: "indented.example.com"},
		{Type: model.EntryDomainSuffix, Value: "example.com"},
		{Type: model.EntryDomainSuffix, Value: "example.com"},
		{Type: model.EntryDomainSuffix, Value: "inline.example.com"},
	}

	if len(entries) != len(want) {
		t.Fatalf("Parse() returned %d entries, want %d\ngot:  %+v\nwant: %+v", len(entries), len(want), entries, want)
	}
	for i, e := range entries {
		if e != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, e, want[i])
		}
	}
}

func TestParseSkipsRegexpAndInclude(t *testing.T) {
	input := "regexp:^example\\.com$\ninclude:other-file\nreal.example.com\n"

	entries, err := Parser{}.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Parse() returned %d entries, want 1 (regexp/include should be skipped): %+v", len(entries), entries)
	}
	if entries[0].Value != "real.example.com" {
		t.Errorf("entries[0].Value = %q, want %q", entries[0].Value, "real.example.com")
	}
}

func TestParseEmptyInput(t *testing.T) {
	entries, err := Parser{}.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Parse(\"\") = %d entries, want 0", len(entries))
	}
}

func TestParseBlankAndCommentOnlyLines(t *testing.T) {
	input := "\n   \n# just a comment\n   # indented comment\n"

	entries, err := Parser{}.Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Parse() = %d entries, want 0: %+v", len(entries), entries)
	}
}
