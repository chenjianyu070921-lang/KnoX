package config

import "time"

type Config struct {
	Mysql struct {
		DSN string `json:"DSN"`
	} `json:"mysql"`
	Redis struct {
		Addr string `json:"Addr"`
	} `json:"redis"`
	Kafka struct {
		Brokers []string `json:"Brokers"`
		Topic   string   `json:"Topic"`
		Group   string   `json:"Group"`
	} `json:"kafka"`
	Ollama struct {
		URL       string `json:"URL"`
		Model     string `json:"Model"`
		Dimension int    `json:"Dimension"`
	} `json:"ollama"`
	Milvus struct {
		Addr        string `json:"Addr"`
		DBName      string `json:"DBName"`
		Collection  string `json:"Collection"`
		VectorField string `json:"VectorField"`
	} `json:"milvus"`
	Consumer struct {
		HandleTimeout  time.Duration `json:"HandleTimeout"`
		LockTTL        time.Duration `json:"LockTTL"`
		LockMaxRetries int           `json:"LockMaxRetries"`
	} `json:"consumer"`
}

// SetDefaults 填充 Ollama/Milvus 的缺省值，避免配置缺省时启动失败。
func (c *Config) SetDefaults() {
	if c.Ollama.URL == "" {
		c.Ollama.URL = "http://localhost:11434"
	}
	if c.Ollama.Model == "" {
		c.Ollama.Model = "bge-m3"
	}
	if c.Ollama.Dimension <= 0 {
		c.Ollama.Dimension = 1024
	}
	if c.Milvus.Addr == "" {
		c.Milvus.Addr = "127.0.0.1:19530"
	}
	if c.Milvus.DBName == "" {
		c.Milvus.DBName = "default"
	}
	if c.Milvus.Collection == "" {
		c.Milvus.Collection = "knox_docs"
	}
	if c.Milvus.VectorField == "" {
		c.Milvus.VectorField = "vector"
	}
}
