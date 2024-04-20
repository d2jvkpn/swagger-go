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

#swag:
#	@if [ ! -d "swagger-go" ]; then \
#	    git clone git@github.com:d2jvkpn/swagger-go.git; \
#	    rm -rf swagger-go/.gitignore swagger-go/.git; \
#	fi
#	bash swagger-go/swag.sh
