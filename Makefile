# go build command
build:
	@echo " >> building binaries"
	@go build -v -o bin/crypto-exchange src/cmd/main.go

# go run command
run: build
	@./bin/crypto-exchange
