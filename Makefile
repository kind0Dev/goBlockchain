build:
	@go build -o bin/go-blockchain
	
run: build
	@./bin/docker

test:
	@go test -v ./...