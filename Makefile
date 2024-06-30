programname=dnsresolver

test:
	@go test -v ./...

build:
	@go build \
		-race \
		-o $(programname) \
		main.go

run: build
	@./$(programname)
