FROM golang:1-alpine AS builder

ENV TZ=${TZ:-Asia/Shanghai}
ARG APP_Name=${APP_Name}
ARG APP_Version=${APP_Version}
ARG BUILD_Region=${BUILD_Region}
ARG GO_ldflags=${GO_ldflags}

LABEL stage=${APP_Name}_builder

RUN if [ "$BUILD_Region" = "cn" ]; then \
    echo "==> BUILD_Region: $BUILD_Region"; \
    sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories; \
    go env -w GOPROXY="https://goproxy.cn,direct"; \
  fi

RUN apk --no-cache update && apk --no-cache upgrade

WORKDIR /app

ADD ./main.go ./go.mod ./go.sum  ./
COPY ./docs ./docs

# in alpine, date doesn't parse %:z
RUN mkdir -p target && \
  go build -o target/main -ldflags="-w -s ${GO_ldflags}" main.go

####
FROM alpine:3

ENV TZ=${TZ:-Asia/Shanghai}
ARG APP_Name=${APP_Name}
ARG APP_Version=${APP_Version}
ARG BUILD_Region=${BUILD_Region}

RUN if [ "$BUILD_Region" = "cn" ]; then \
    echo "==> BUILD_Region: $BUILD_Region"; \
    sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories; \
  fi

RUN apk --no-cache update && \
  apk --no-cache upgrade && \
  apk --no-cache add tzdata
  # curl

RUN mkdir -p /app/target
WORKDIR /app

COPY --from=builder /app/target/main /app/target/main

EXPOSE 3056
CMD ["./target/main", "-release", "-http.addr=:3056"]
