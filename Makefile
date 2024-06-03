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
	bash swag.sh app-swagger

run:
	bash swag.sh app-swagger
	./target/APP-swagger -swagger.title="app swagger"

run-with-config:
	bash swag.sh app-swagger
	./target/app-swagger -swagger.title="app swagger" -config=configs/swagger.yaml

image-cn:
	BUILD_Region=cn DOCKER_Tag=dev bash deployments/docker_build.sh main

#swag-update:
#	@if [ ! -d "swagger-go" ]; then \
#	    git clone git@github.com:d2jvkpn/swagger-go.git /tmp/swagger-go; \
#	    rsync -arvP --exclude .git /tmp/swagger-go ./; \
#	fi
#	bash swagger-go/swag.sh app-swagger

#swag-run:
#	bash swagger-go/swag.sh app-swagger
#	./target/app-swagger -swagger.title="app swagger"
#	# ./target/app-swagger -swagger.title="app swagger" -config=configs/swagger.yaml
