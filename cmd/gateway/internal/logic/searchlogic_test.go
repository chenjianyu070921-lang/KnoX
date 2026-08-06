package logic

import (
	"strings"
	"testing"

	"github.com/chenjianyu070921-lang/KnoX/cmd/gateway/internal/types"
	"github.com/cloudwego/eino/schema"
)

func TestBuildSearchResults(t *testing.T) {
	docs := []*schema.Document{
		{ID: "doc_1", Content: "first", MetaData: map[string]any{"_score": 0.87}},
		{ID: "doc_2", Content: "second"},
	}

	results := buildSearchResults(docs)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].DocId != "doc_1" || results[0].Content != "first" {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	if results[0].Score != 0.87 {
		t.Fatalf("expected score 0.87, got %v", results[0].Score)
	}
	if results[1].Score != 0 {
		t.Fatalf("expected missing score to map to 0, got %v", results[1].Score)
	}
}

func TestNormalizeTopK(t *testing.T) {
	cases := []struct {
		requested int
		def       int
		max       int
		want      int
	}{
		{0, 5, 20, 5},
		{-1, 5, 20, 5},
		{3, 5, 20, 3},
		{99, 5, 20, 20},
		{10, 8, 30, 10},
	}
	for _, c := range cases {
		if got := normalizeTopK(c.requested, c.def, c.max); got != c.want {
			t.Fatalf("normalizeTopK(%d, %d, %d) = %d, want %d", c.requested, c.def, c.max, got, c.want)
		}
	}
}

func TestValidateSearchRequest(t *testing.T) {
	if err := validateSearchRequest(&types.SearchReq{Query: ""}); err == nil {
		t.Fatal("expected empty query to fail")
	}
	if err := validateSearchRequest(&types.SearchReq{Query: "  "}); err == nil {
		t.Fatal("expected whitespace query to fail")
	}
	if err := validateSearchRequest(&types.SearchReq{Query: "知识库"}); err != nil {
		t.Fatalf("expected valid query to pass: %v", err)
	}
	if err := validateSearchRequest(&types.SearchReq{Query: strings.Repeat("q", 501)}); err == nil {
		t.Fatal("expected oversized query to fail")
	}
}
