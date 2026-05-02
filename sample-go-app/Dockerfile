FROM golang:1.24 AS builder

WORKDIR /

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN --mount=type=cache,target="/root/.cache/go-build" \
  # -s disables symbol table
  # -w disables DWARF generation
  # both reduce the final binary size
  CGO_ENABLED=0 go build -ldflags="-s -w -extldflags=-static" -o /sample-go-app

FROM gcr.io/distroless/static-debian12:nonroot

# ARG injection to set the VERSION envvar, this is more efficient than injecting the info into the go binary, as it doesn't require rebuilding the binary for each version change.
# The VERSION envvar can be used by the app to display the version or for other purposes.
ARG VERSION="unset"
ENV VERSION=$VERSION

WORKDIR /

COPY --from=builder /sample-go-app /sample-go-app

ENTRYPOINT [ "/sample-go-app" ]
