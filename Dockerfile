FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

COPY . $GOPATH/src/crmsrvg/

WORKDIR $GOPATH/src/crmsrvg
RUN go mod tidy
# RUN go mod download
RUN mkdir /app
RUN go build -tags migrate -o /app/crm ./cmd/api-server/

FROM alpine:latest

RUN mkdir /app
COPY --from=builder /app/crm /app/crm
WORKDIR /app 
COPY ./config/config.yaml ./config/
COPY ./docs/swagger.* ./docs/
COPY ./migrations/* ./migrations/
COPY .env .

CMD ["/app/crm", "-config=./config/config.yaml"]
