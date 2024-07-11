#build stage
FROM golang:alpine AS builder

RUN apk update && apk add --no-cache libc-dev make
RUN apk add --virtual build-dependencies build-base
WORKDIR /go/src/app
# Copy the Go application
COPY . .
COPY ./infra/scripts/startup.sh build/
# Build the Go application
RUN go build -o build/faucet


#final stage
FROM alpine:latest
RUN addgroup -S nuklai && adduser -S nuklai -G nuklai
COPY --from=builder --chown=nuklai /go/src/app/build /app
USER nuklai
RUN chmod +x /app/startup.sh
RUN ls -la /app
ENTRYPOINT [ "/app/startup.sh" ]
CMD ["/app/faucet"]
LABEL Name=faucetrpc
EXPOSE 10591
