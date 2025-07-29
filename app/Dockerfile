# Stage 1: Build
FROM golang:1.24-alpine3.20 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o nmock main.go

# Stage 2: Runtime
FROM alpine:3.20
WORKDIR /app
RUN apk add ca-certificates --no-cache
COPY --from=builder /app/nmock .
COPY --from=builder /app/config.json .
USER 1001
EXPOSE 9000
CMD [ "/app/nmock" ]