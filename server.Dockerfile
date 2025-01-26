FROM golang:1.23.1-alpine as modules
COPY go.mod go.sum /modules/
WORKDIR /modules
RUN go mod download

FROM golang:1.23.1-alpine as builder
COPY --from=modules /go/pkg /go/pkg
COPY . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /bin ./cmd/server

FROM alpine
COPY lintang /lintang
COPY docs_store.db .
COPY --from=builder /bin /bin
CMD ["/bin/server"]
