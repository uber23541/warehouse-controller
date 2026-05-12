FROM golang AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o warehouse-controller ./server

FROM --platform=linux/amd64 alpine:3.20
WORKDIR /app

COPY --from=builder /app/warehouse-controller .
COPY --from=builder /app/db/migrations ./db/migrations
EXPOSE 8080

CMD ["./warehouse-controller"]