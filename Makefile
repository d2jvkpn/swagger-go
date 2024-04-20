#/bin/make
# include envfile
# export $(shell sed 's/=.*//' envfile)

current = $(shell pwd)

build_time = $(shell date +'%FT%T.%N%:z')
git_repository = $(shell git config --get remote.origin.url)
git_branch = $(shell git rev-parse --abbrev-ref HEAD)
git_commit_id = $(shell git rev-parse --verify HEAD)
git_commit_time = $(shell git log -1 --format="%at" | xargs -I{} date -d @{} +%FT%T%:z)

go:
	go mod tidy
	if [ -d vendor ]; then go mod vendor; fi
	go fmt ./...
	go vet ./...

build:
	bash swag.sh
	mkdir -p target

	@go build -ldflags="-w -s \
	  -X main.build_time=$(build_time) \
	  -X main.git_repository=$(git_repository) \
	  -X main.git_branch=$(git_branch) \
	  -X main.git_commit_id=$(git_commit_id) \
	  -X main.git_commit_time=$(git_commit_time)" \
	  -o target/swagger-go main.go

	ls -alh target/

run:
	./target/swagger-go

#swag:
#	@if [ ! -d "swagger-go" ]; then \
#	    git clone git@github.com:d2jvkpn/swagger-go.git /tmp/swagger-go; \
#	    rsync -arvP --exclude .git /tmp/swagger-go ./; \
#	    rm -rf /tmp/swagger-go; \
#	fi
#	bash swagger-go/swag.sh $(shell pwd)
