FROM golang:1.17.10-alpine3.15 as builder
RUN  	 mkdir /build && \
	 cd /build
WORKDIR "/build"
COPY . ./
RUN 	go get github.com/tidwall/gjson 
RUN 	go build

FROM alpine:3.15 
COPY --from=builder /build/currency-converter .
ENTRYPOINT [ "./currency-converter" ]
