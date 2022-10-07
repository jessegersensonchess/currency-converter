FROM golang:1.19-alpine3.16 as builder
RUN  	 mkdir /build && \
	 cd /build
WORKDIR "/build"
COPY . ./
RUN 	go get github.com/go-redis/redis/v8 && \
 	go build

FROM alpine:3.16
COPY --from=builder /build/currency-converter .
ENTRYPOINT [ "./currency-converter" ]
