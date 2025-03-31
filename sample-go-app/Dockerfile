FROM golang:1.24 AS builder

ARG VERSION="unset"

WORKDIR /

COPY go.mod go.sum ./

RUN --mount=type=cache,target="/root/.cache/go-build" \
    --mount=type=cache,target="/root/.cache/go-build" \
  go mod download

COPY . .

RUN --mount=type=cache,target="/root/.cache/go-build" \
  --mount=type=cache,target="/root/.cache/go-build" \
  CGO_ENABLED=0 go build -ldflags="-X 'main.Version=${VERSION}' -extldflags=-static"

FROM alpine:3.12

WORKDIR /

COPY --from=builder /sample-go-app /sample-go-app

ENTRYPOINT [ "sample-go-app" ]
