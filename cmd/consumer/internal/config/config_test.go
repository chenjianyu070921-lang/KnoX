package config

import "testing"

func TestConfigSetDefaults(t *testing.T) {
	var c Config
	c.SetDefaults()

	if c.Ollama.URL != "http://localhost:11434" {
		t.Fatalf("unexpected ollama url: %s", c.Ollama.URL)
	}
	if c.Ollama.Model != "bge-m3" {
		t.Fatalf("unexpected ollama model: %s", c.Ollama.Model)
	}
	if c.Ollama.Dimension != 1024 {
		t.Fatalf("unexpected ollama dimension: %d", c.Ollama.Dimension)
	}
	if c.Milvus.Addr != "127.0.0.1:19530" {
		t.Fatalf("unexpected milvus addr: %s", c.Milvus.Addr)
	}
	if c.Milvus.DBName != "default" {
		t.Fatalf("unexpected milvus db name: %s", c.Milvus.DBName)
	}
	if c.Milvus.Collection != "knox_docs" {
		t.Fatalf("unexpected milvus collection: %s", c.Milvus.Collection)
	}
	if c.Milvus.VectorField != "vector" {
		t.Fatalf("unexpected milvus vector field: %s", c.Milvus.VectorField)
	}
}
