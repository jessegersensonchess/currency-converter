FROM golang:1.17.10-alpine3.15
RUN  	 mkdir /build && \
	 cd /build
WORKDIR "/build"
COPY . ./
RUN 	go get github.com/tidwall/gjson && \
	go build -o currency-converter main.go
ENTRYPOINT [ "./currency-converter" ]
