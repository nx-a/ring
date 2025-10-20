FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build  -ldflags="-s -w" -buildvcs=false -a -installsuffix cgo -o app .


FROM alpine:latest
WORKDIR /opt/ring

COPY --from=builder /app/app .
EXPOSE 80
CMD ["./app"]