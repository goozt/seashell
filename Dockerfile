FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /app/bin/seashell .

# ---

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/bin/seashell /app/seashell

# DB directory is mounted as a volume at runtime.
RUN mkdir -p /app/db

EXPOSE 8080
ENTRYPOINT ["/app/seashell", "serve"]
