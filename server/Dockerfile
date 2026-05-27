FROM golang:1.26-alpine AS builder

WORKDIR /app

# Use Chinese Go module proxy for faster downloads in China
ENV GOPROXY=https://goproxy.cn,direct

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -a -o learncode ./cmd/learncode

FROM alpine:3.21

RUN apk add --no-cache ca-certificates docker-cli

# Add Accesser root CA for Wikipedia domain fronting (via proxy)
COPY accesser/CERT/root.crt /usr/local/share/ca-certificates/accesser.crt
RUN update-ca-certificates

WORKDIR /app

COPY --from=builder /app/learncode /app/learncode
COPY --from=builder /app/config.yaml /app/config.yaml
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/prompts /app/prompts

EXPOSE 8080

CMD ["./learncode"]
