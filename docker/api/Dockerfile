FROM golang:1.25-alpine AS builder
WORKDIR /app
# Cache module download
COPY go.mod go.sum ./
RUN go mod download
# Copy source and build
COPY . .
RUN GOMAXPROCS=1 CGO_ENABLED=0 GOOS=linux go build -p=1 -o /api ./cmd/api

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /api .
EXPOSE 8080
ENTRYPOINT ["/app/api"]
