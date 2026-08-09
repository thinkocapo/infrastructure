FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o infrastructure-monitor .

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/infrastructure-monitor .
CMD ["./infrastructure-monitor"]
