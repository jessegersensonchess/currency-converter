FROM golang:1.19-alpine3.21 as builder
RUN  mkdir /build && \
	 cd /build
WORKDIR "/build"
COPY . ./
RUN	 go get github.com/go-redis/redis/v8 && \
     go get github.com/tidwall/gjson && \
 	 go build

FROM alpine:3.21
COPY --from=builder /build/currency-converter .
ENTRYPOINT [ "./currency-converter" ]
