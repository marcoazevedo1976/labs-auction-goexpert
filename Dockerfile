FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .

WORKDIR /app/cmd/auction
RUN CGO_ENABLED=0 GOOS=linux go build -o auction .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/cmd/auction/auction .
# COPY --from=builder /app/cmd/auction/.env .
COPY wait-for-it.sh .
RUN chmod +x /app/wait-for-it.sh

EXPOSE 8080
CMD ["/app/auction"]