package logic

import (
	"testing"

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
