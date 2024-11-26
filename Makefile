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

	app_name=app-swagger bash swag.sh false > /dev/null

build:
	app_name=app-swagger bash swag.sh false > /dev/null

run:
	app_name=app-swagger bash swag.sh true > /dev/null
	./target/app-swagger -swagger.title="app swagger" -http.addr=:3066

run-with-config:
	app_name=app-swagger bash swag.sh true > /dev/null
	./target/app-swagger -swagger.title="app swagger" -config=configs/swagger.yaml

image-dev:
	BUILD_Region=cn DOCKER_Push=false DOCKER_Tag=dev \
	  bash deployments/build.sh dev

deploy-dev:
	bash deployments/compose.sh dev 3067

#build-swag:
#	@if [ ! -d "bin/swagger-go" ]; then \
#	    git clone git@github.com:d2jvkpn/swagger-go.git /tmp/swagger-go; \
#	    mkdir -p bin; \
#	    rsync -arvP --exclude .git /tmp/swagger-go ./bin/; \
#	fi
#	app_name=app-swagger bash bin/swagger-go/swag.sh false

#run-swag:
#	app_name=app-swagger bash bin/swagger-go/swag.sh true
#	./target/app-swagger -swagger.title="app swagger"
#	# ./target/app-swagger -swagger.title="app swagger" -config=configs/swagger.yaml -http.addr=:3066
