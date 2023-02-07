.PHONY: generate
generate:
	@go run cmd/gomergetypes/main.go --config ./example/example-gomergetypes.yml

.PHONY: test
test:
	@go test -v ./...
