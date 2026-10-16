FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/recommendation-service ./cmd

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/recommendation-service /recommendation-service

ENV GIN_MODE=release \
    PORT=3000
EXPOSE 3000

ENTRYPOINT ["/recommendation-service"]
