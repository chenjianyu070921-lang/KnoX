run:
	go run ./cmd/gateway -f cmd/gateway/etc/doc.yaml
run-consumer:
	go run ./cmd/consumer -f cmd/consumer/etc/config.yaml
goctl:
	goctl api go -api api/doc.api -dir cmd/gateway
