.PHONY: test
test:
	go test -tags exclude_frontend,unit ./...

.PHONY: test-race
test-race:
	CGO_ENABLED=1 go test -race -tags exclude_frontend,unit ./...

.PHONY: lint
lint:
	golangci-lint run -c .golangci.yml
