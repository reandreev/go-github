FROM golang:1.22.5

WORKDIR /app

RUN mkdir -p bin cmd/go-github internal/routes

ADD go.mod ./
ADD go.sum ./

COPY cmd/go-github ./cmd/go-github
COPY internal/routes ./internal/routes

RUN go build -o bin ./...

EXPOSE 8080

CMD ["./bin/go-github"]
