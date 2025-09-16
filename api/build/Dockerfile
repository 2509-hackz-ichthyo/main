FROM golang:1.24.1-alpine

WORKDIR /app

COPY . ./

RUN go mod download

RUN go build -o main /app/main.go

EXPOSE 3000

CMD /app/main