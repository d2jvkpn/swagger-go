### A server that provides Swagger API documentation for Golang projects
---

#### P1. Links
- https://swagger.io/
- https://github.com/swaggo
- https://github.com/swaggo/swag?tab=readme-ov-file#how-to-use-it-with-gin
- https://github.com/swaggo/http-swagger
- https://www.linkedin.com/pulse/swagger-go-pragmatic-guide-phuong-le-2gbhc

#### P2. Append to Makefile
```bash

cat >> Makefile <<'EOF'

build-swag:
	@if [ ! -d "swagger-go" ]; then \
	    git clone git@github.com:d2jvkpn/swagger-go.git /tmp/swagger-go; \
	    mkdir -p bin; \
	    rsync -arvP --exclude .git /tmp/swagger-go ./bin/; \
	fi
	app_name=app-swagger bash bin/swagger-go/swag.sh false

run-swag:
	app_name=app-swagger bash bin/swagger-go/swag.sh app-swagger true
	./target/app-swagger -swagger.title "app swagger"
#	# ./target/app-swagger -swagger.title "app swagger" -config=configs/swagger.yaml

EOF

```
