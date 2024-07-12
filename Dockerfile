FROM golang:1.22.5

WORKDIR /app

ADD go.mod ./
ADD go.sum ./
ADD main.go ./

RUN go build

EXPOSE 8080

CMD ["./go-github"]
