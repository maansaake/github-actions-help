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
  # -s disables symbol table
  # -w disables DWARF generation
  # both reduce the final binary size
  CGO_ENABLED=0 go build -ldflags="-s -w -X 'main.Version=${VERSION}' -extldflags=-static" -o /sample-go-app ./cmd

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=builder /sample-go-app /sample-go-app

ENTRYPOINT [ "/sample-go-app" ]
