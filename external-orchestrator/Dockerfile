# Stage 1: Builder
FROM golang:1.25.4-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /app/main .

# Stage 2: Runner
FROM alpine:3.22.3
WORKDIR /app
RUN apk --no-cache add curl
COPY --from=builder /app/main .
EXPOSE 8080

# Run
CMD ["./main"]