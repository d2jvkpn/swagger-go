#/bin/make
# include envfile
# export $(shell sed 's/=.*//' envfile)

current = $(shell pwd)

# build_time = $(shell date +'%FT%T.%N%:z')
build_time = $(shell date +'%FT%T%:z')
git_repository = $(shell git config --get remote.origin.url)
git_branch = $(shell git rev-parse --abbrev-ref HEAD)
git_commit_id = $(shell git rev-parse --verify HEAD)
git_commit_time = $(shell git log -1 --format="%at" | xargs -I{} date -d @{} +%FT%T%:z)

lint:
	go mod tidy
	if [ -d vendor ]; then go mod vendor; fi
	go fmt ./...
	go vet ./...

build:
	bash swag.sh APP-swagger

run:
	bash swag.sh APP-swagger
	./target/APP-swagger -swagger.title "APP Swagger"

image-cn:
	BUILD_Region=cn DOCKER_Tag=dev bash deployments/docker_build.sh main

#swag-update:
#	@if [ ! -d "swagger" ]; then \
#	    git clone git@github.com:d2jvkpn/swagger-go.git /tmp/swagger-go; \
#	    rsync -arvP --exclude .git /tmp/swagger-go ./swagger; \
#	fi
#	bash swagger/swag.sh APP-swagger

#swag-run:
#	bash swagger/swag.sh
#	./swagger/target/APP-swagger
