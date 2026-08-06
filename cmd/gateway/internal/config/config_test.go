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
	if c.Retrieval.DefaultTopK != 5 {
		t.Fatalf("unexpected default topK: %d", c.Retrieval.DefaultTopK)
	}
	if c.Retrieval.MaxTopK != 20 {
		t.Fatalf("unexpected max topK: %d", c.Retrieval.MaxTopK)
	}
}

func TestConfigSetDefaultsKeepsExplicitValues(t *testing.T) {
	c := Config{}
	c.Ollama.URL = "http://ollama:11434"
	c.Ollama.Model = "other-model"
	c.Ollama.Dimension = 768
	c.Milvus.Addr = "milvus:19530"
	c.Milvus.DBName = "custom-db"
	c.Milvus.Collection = "custom_coll"
	c.Milvus.VectorField = "vec_field"
	c.Retrieval.DefaultTopK = 8
	c.Retrieval.MaxTopK = 30

	c.SetDefaults()

	if c.Ollama.URL != "http://ollama:11434" || c.Ollama.Model != "other-model" || c.Ollama.Dimension != 768 {
		t.Fatalf("explicit ollama config was overwritten: %+v", c.Ollama)
	}
	if c.Milvus.Addr != "milvus:19530" || c.Milvus.DBName != "custom-db" ||
		c.Milvus.Collection != "custom_coll" || c.Milvus.VectorField != "vec_field" {
		t.Fatalf("explicit milvus config was overwritten: %+v", c.Milvus)
	}
	if c.Retrieval.DefaultTopK != 8 || c.Retrieval.MaxTopK != 30 {
		t.Fatalf("explicit retrieval config was overwritten: %+v", c.Retrieval)
	}
}
