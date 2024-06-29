test:
	@go test -v ./...

run:
	@go build main.go
	@./main
