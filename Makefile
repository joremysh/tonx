dependencies:
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

generate:
	go generate ./...

unit-test:
	go test -v ./...

lint:
	#golangci-lint run --timeout 60s
	docker build --rm -t golangci-lint-check -f Dockerfile.lint .

up:
	docker-compose up -d

down:
	docker-compose down
