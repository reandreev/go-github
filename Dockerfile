FROM golang:1.22.5

WORKDIR /app

RUN git clone https://github.com/reandreev/go-github.git .
RUN go build

EXPOSE 8080

CMD ["./go-github"]
