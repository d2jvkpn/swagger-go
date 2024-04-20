go:
	go mod tidy
	if [ -d vendor ]; then go mod vendor; fi
	go fmt ./...
	go vet ./...

build:
	bash swag.sh
	mkdir -p target
	go build -o target/swagger-go main.go
	ls -alh target/

run:
	bash swag.sh
	go run main.go
