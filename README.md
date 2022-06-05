Currency converter
=======================
Converts between ISO 4217 currency codes. Takes 2 or 3 arguments. The last arg is opitonal. 

Usage: go build && ./currency-converter usd eur 100

```
docker build -t currency/converter:latest .
```

Convert USD to Euro

```
docker run -it --rm currency/converter:latest usd eur 100
```

To do
------------
 - add error handling
 - add logging
 - add docker-compose.yml
 - move config to config management system
 - use structs and std library in place of gjson
 - get a code review
 - add RESTful api
