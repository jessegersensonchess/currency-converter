Currency converter
=======================
Converts between ISO 4217 currency codes. Takes 2 or 3 arguments. The last arg is opitonal. 

Script attempts to connect to, and retrieve data from, redis over port 6379.

Usage: 
--------

```./currency-converter usd eur 100```

Build:
--------

```
go build 
# or, build in docker
docker build -t currency/converter:latest .
```

Example
--------
Converts 100 USD to Euro

```
docker run -it --rm currency/converter:latest usd eur 100
```

To do
------------
 - add error handling
 - add logging
 - add docker-compose.yml
 - move config to config management system
 - add RESTful api
