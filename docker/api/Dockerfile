FROM golang:1.16.5-alpine3.13

WORKDIR /app

COPY go.mod .

COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o ./out/image-generaor-api .

EXPOSE 80

CMD ["./out/image-generaor-api"]
