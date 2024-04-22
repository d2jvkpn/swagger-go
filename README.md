### A server that provides Swagger API documentation for Golang projects
---

#### P1. Links
- https://swagger.io/
- https://github.com/swaggo
- https://github.com/swaggo/swag?tab=readme-ov-file#how-to-use-it-with-gin
- https://github.com/swaggo/http-swagger

#### P2. Append to Makefile
```bash

cat >> Makefile <<'EOF'

swag-update:
	@if [ ! -d "swagger-go" ]; then \
	    git clone git@github.com:d2jvkpn/swagger-go.git /tmp/swagger-go; \
	    rsync -arvP --exclude .git /tmp/swagger-go ./; \
	    rm -rf /tmp/swagger-go; \
	fi
	bash swagger-go/swag.sh

swag-run:
	bash swagger-go/swag.sh
	./swagger-go/target/swagger-go

EOF

```
