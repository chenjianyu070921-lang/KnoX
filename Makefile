run:
	go run ./cmd/gateway -f cmd/gateway/etc/doc.yaml
run-consumer:
	go run ./cmd/consumer -f cmd/consumer/etc/config.yaml
run-reporter:
	go run ./cmd/reporter -f cmd/reporter/etc/reporter.yaml
goctl:
	goctl api go -api api/doc.api -dir cmd/gateway
run-web:
	cd web && npm run dev