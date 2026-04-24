# syntax=docker/dockerfile:1.7

# ------------ build stage ------------
FROM golang:1.23-alpine AS build
WORKDIR /src

ARG VERSION=docker
ARG COMMIT=unknown
ARG DATE

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build \
      -trimpath \
      -ldflags "-s -w \
        -X github.com/pluginmd/haravan-cli/internal/build.Version=${VERSION} \
        -X github.com/pluginmd/haravan-cli/internal/build.Commit=${COMMIT} \
        -X github.com/pluginmd/haravan-cli/internal/build.Date=${DATE}" \
      -o /out/haravan-cli .

# ------------ runtime stage ------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tini \
    && addgroup -S app && adduser -S -G app app

COPY --from=build /out/haravan-cli /usr/local/bin/haravan-cli
USER app
WORKDIR /home/app

EXPOSE 4567
ENTRYPOINT ["/sbin/tini", "--", "/usr/local/bin/haravan-cli"]
CMD ["mcp", "serve", "--http", "--addr=:4567"]
