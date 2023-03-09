FROM golang:1.20.2-alpine3.17 as builder

WORKDIR /build

COPY . .

ENV CGO_ENABLED=0
RUN go mod download
RUN go build -o main main.go

FROM alpine:3.17.2

RUN apk update && apk upgrade && \
    apk add --no-cache bash ca-certificates

COPY --from=builder build/main .

CMD ["./main"]