programname=dnsresolver

test:
	@go test -v ./...

build:
	@go build -v \
		-race \
		-o $(programname) \
		main.go

run: build
	@./$(programname)
