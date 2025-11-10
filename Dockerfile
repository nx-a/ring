FROM alpine:latest

WORKDIR /opt/ring

COPY ./app .
EXPOSE 9080 7888
CMD ["./app"]