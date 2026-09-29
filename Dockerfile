# syntax=docker/dockerfile:1

FROM golang:1.27-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/massmaker-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/massmaker-api /app/massmaker-api
COPY --from=builder /src/migrations /app/migrations
EXPOSE 8080
ENTRYPOINT ["/app/massmaker-api"]
