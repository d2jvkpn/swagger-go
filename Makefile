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

build-local:
	bash swag.sh APP_swagger-go

run:
	bash swag.sh
	./target/swagger-go -swagger.title "Swagger for APP"

build-image_cn:
	BUILD_Region=cn DOCKER_Tag=dev bash deployments/docker_build.sh main

#swag-update:
#	@if [ ! -d "swagger-go" ]; then \
#	    git clone git@github.com:d2jvkpn/swagger-go.git /tmp/swagger-go; \
#	    rsync -arvP --exclude .git /tmp/swagger-go ./; \
#	    rm -rf /tmp/swagger-go; \
#	fi
#	bash swagger-go/swag.sh

#swag-run:
#	bash swagger-go/swag.sh
#	./swagger-go/target/swagger-go
