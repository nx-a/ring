FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build  -ldflags="-s -w" -buildvcs=false -a -installsuffix cgo -o app .


FROM alpine:latest

RUN apk add --no-cache ca-certificates && \
    # Создаем скрипт для настройки sysctl
    echo '#!/bin/sh' > /app/setup.sh && \
    echo 'sysctl -w net.core.rmem_max=8388608' >> /app/setup.sh && \
    echo 'sysctl -w net.core.wmem_max=8388608' >> /app/setup.sh && \
    echo 'sysctl -w net.core.rmem_default=2097152' >> /app/setup.sh && \
    echo 'sysctl -w net.core.wmem_default=2097152' >> /app/setup.sh && \
    echo 'exec /app/quic-server' >> /app/setup.sh && \
    chmod +x /app/setup.sh \

WORKDIR /opt/ring

COPY --from=builder /app/app .
EXPOSE 80 7888
CMD ["./app"]