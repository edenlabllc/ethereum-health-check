# Ethereum Node Health Check Microservice

Microservice that checks ethereum node health, by comparing node block number with block number from [etherscan](https://etherscan.io)
```
$ curl -H 'Content-Type: application/json'  http://0.0.0.0:8080/ethealth/check
{"health":true}

$ curl -H 'Content-Type: application/json'  http://0.0.0.0:8080/ethealth/check
{"health":false, diff: 10}
```

## Install

```
go get github.com/micro/go-micro
```
## Required ENV variables

|Name   |Type   |Description    |
|---	|---	|---	|
|NODE_ADDR  	|URL   	| URL for node to connect, with schema  	|
|MAXBLOCKDIFFERENCE   	|int   	| Number of blocks difference between healthchecked node and etherscan api that count unhealthy |
|ETHERSCAN_PREFIX   	|string   	| Testnet name    	|
|ETHERSCAN_API_KEY   	|string   	| Etherscan api key. https://etherscan.io/apis |

## Usage

```
MICRO_REGISTRY=mdns micro api --handler=rpc
MICRO_REGISTRY=mdns NODE_ADDR= MAXBLOCKDIFFERENCE=3 ETHERSCAN_PREFIX=rinkeby ETHERSCAN_API_KEY= go run main.go

curl -H 'Content-Type: application/json'  http://localhost:8080/ethealth/check
```

## Deploy

Setup proper ENV variables
```
docker-compose up -d
```

## Contributing

PRs accepted.

## License

MIT © Edenlabllc http://edenlab.com.ua