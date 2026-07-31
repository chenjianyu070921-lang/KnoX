package config

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
		URL   string `json:"URL"`
		Model string `json:"Model"`
	} `json:"ollama"`
	Milvus struct {
		Addr string `json:"Addr"`
	} `json:"milvus"`
}
