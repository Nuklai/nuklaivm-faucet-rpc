#build stage
FROM golang:alpine AS builder

RUN apk update && apk add --no-cache libc-dev make bash
RUN apk add --virtual build-dependencies build-base
WORKDIR /go/src/app
# Copy the Go application
COPY . .
COPY ./infra/scripts/startup.sh build/
# Build the Go application
RUN go build -o build/faucet


#final stage
FROM alpine:latest
RUN apk update && apk add --no-cache bash
RUN addgroup -S nuklai && adduser -S nuklai -G nuklai
COPY --from=builder --chown=nuklai /go/src/app/build /app
USER nuklai
RUN chmod a+x /app/startup.sh
ENTRYPOINT [ "/app/startup.sh" ]
RUN ls -la /app
RUN cat /app/startup.sh
LABEL Name=faucetrpc
EXPOSE 10591
WORKDIR /app
CMD [ "/app/faucet" ]
