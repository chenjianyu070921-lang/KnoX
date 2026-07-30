// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Mysql struct {
		DSN string `json:"DSN"`
	} `json:"mysql"`
	Redis struct {
		Addr string `json:"addr"`
	} `json:"redis"`
	Kafka struct {
		Brokers []string `json:"brokers"`
		Topic   string   `json:"topic"`
	} `json:"kafka"`
	Qiniu struct {
		AccessKey string `json:"accessKey"`
		SecretKey string `json:"secretKey"`
		Bucket    string `json:"bucket"`
		Region    string `json:"region"` // 区域ID，如 z0=华东, z1=华北, z2=华南, na0=北美, as0=东南亚
		Domain    string `json:"domain"` // 外链默认域名，如 https://xxx.clouddn.com
	} `json:"qiniu"`
	ARK struct {
		APIKey  string `json:"apiKey"`
		ModelID string `json:"modelId"`
		BaseURL string `json:"baseUrl"`
	} `json:"ark"`
}
