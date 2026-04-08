FROM golang:1.23-alpine AS builder
WORKDIR /app
# Cache module download
COPY go.mod go.sum ./
RUN go mod download
# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /api .
EXPOSE 8080
ENTRYPOINT ["/app/api"]
