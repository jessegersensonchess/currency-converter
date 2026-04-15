FROM golang:1.25-alpine AS builder
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'currency_converter/version.Version=v$(cat VERSION_NUMBER)'" -o /go/currency_converter ./cmd/currency
RUN apk add --no-cache ca-certificates

FROM scratch
COPY --from=builder /go/currency_converter .
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENTRYPOINT [ "./currency_converter" ]
