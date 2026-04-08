FROM golang:1.23-alpine AS builder
ARG WORKER_TYPE
WORKDIR /app
# Cache module download — only re-runs when go.mod/go.sum change
COPY go.mod go.sum ./
RUN go mod download
# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /worker ./cmd/workers/${WORKER_TYPE}

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /worker .
ENTRYPOINT ["/app/worker"]
