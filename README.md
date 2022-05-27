Currency converter
=======================
Converts from one currency into another. Takes 2 or 3 arguments. The last arg is opitonal. 

Usage: go run main.go usd eur 100

```
docker build -t currency/converter:latest .
```

Convert USD to Euro

```
docker run -it --rm currency:latest usd eur
```

To do
------------
 - add error handling
 - add caching
 - use structs and std library in place of gjson
 - shrink docker image
