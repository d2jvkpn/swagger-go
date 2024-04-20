#/bin/make
# include envfile
# export $(shell sed 's/=.*//' envfile)

current = $(shell pwd)

build_time = $(shell date +'%FT%T.%N%:z')
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

	@go build -ldflags="-w -s -X main.BUILD_Time=$(build_time) \
	  -X main.GIT_Branch=$(git_branch) \
	  -X main.GIT_CommitId=$(git_commit_id) \
	  -X main.GIT_CommitTime=$(git_commit_time)" -o target/swagger-go main.go

	ls -alh target/

run:
	bash swag.sh
	go run main.go

#swag:
#	@if [ ! -d "swagger-go" ]; then \
#	    git clone git@github.com:d2jvkpn/swagger-go.git; \
#	    rm -rf swagger-go/.gitignore swagger-go/.git; \
#	fi
#	bash swagger-go/swag.sh $(shell pwd)
