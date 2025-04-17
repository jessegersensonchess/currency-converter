FROM golang:1.19-alpine3.21 as builder
RUN  mkdir /build && \
	 cd /build
WORKDIR "/build"
COPY . ./
# todo: get build version in here
RUN	 go build

FROM alpine:3.21
COPY --from=builder /build/currency-converter .
ENTRYPOINT [ "./currency-converter" ]
