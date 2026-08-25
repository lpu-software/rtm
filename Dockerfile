FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o rtm ./cmd/rtm

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/rtm .
EXPOSE 7860
CMD ["./rtm", "serve", "-port", "7860"]
