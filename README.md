<div align="center">
  <img src="https://raw.githubusercontent.com/goipay/goipay.github.io/refs/heads/master/static/img/goipay-logo-with-name.svg" alt="Logo" width="400" height="100">

  ![Build Status](https://img.shields.io/github/actions/workflow/status/goipay/goipay/cd.yml)
  ![Version](https://img.shields.io/github/v/release/goipay/goipay)
  ![Docker Pulls](https://img.shields.io/docker/pulls/chekist32/goipay)
  ![License](https://img.shields.io/github/license/goipay/goipay)
</div>

## Description

A lightweight crypto payment processor microservice, written in Golang, designed for creating and processing cryptocurrency invoices via gRPC.
### Supported Crypto
- XMR
- BTC
- LTC
- ETH (USDT, USDC, DAI, WBTC, UNI, LINK, AAVE, CRV, MATIC, SHIB, BNB, ATOM, ARB)
- BNB (BSC-USD, USDC, DAI, BUSD, WBTC, BTCB, UNI, LINK, AAVE, MATIC, SHIB, ATOM, ARB, ETH, XRP, ADA, TRX, DOGE, LTC, BCH, TWT, AVAX, CAKE)

## Getting Started
### Prerequisites
- Go ≥ 1.22
- PostgreSQL ≥ 12

### Installation
#### Docker
- Clone the repo
  ```sh
  git clone https://github.com/goipay/goipay.git
  ```
- Inside the root dir create and populate ```.env``` file on the base of ```.env.example``` file
  ```ini
  SERVER_HOST=0.0.0.0
  SERVER_PORT=3000
  
  SERVER_TLS_MODE=tls
  SERVER_TLS_CA=/app/cert/server/ca.crt
  SERVER_TLS_CERT=/app/cert/server/server.crt
  SERVER_TLS_KEY=/app/cert/server/server.key
  
  # As for now, only PostgreSQL is supported
  DATABASE_HOST=db
  DATABASE_PORT=5432
  DATABASE_USER=postgres
  DATABASE_PASS=postgres
  DATABASE_NAME=goipay_db
  
  XMR_DAEMON_URL=http://node.monerodevs.org:38089
  XMR_DAEMON_USER=
  XMR_DAEMON_PASS=
  
  BTC_DAEMON_URL=http://localhost:38332
  BTC_DAEMON_USER=user
  BTC_DAEMON_PASS=pass
  
  LTC_DAEMON_URL=http://localhost:18444
  LTC_DAEMON_USER=user
  LTC_DAEMON_PASS=pass

  ETH_DAEMON_URL=https://ethereum.publicnode.com

  BNB_DAEMON_URL=https://bsc-dataseed.binance.org
  ```
- Inside the root dir you can find an example ```docker-compose.yml``` file. For testing purposes can be run without editing.
  ```sh
  docker compose up
  ```
#### Native
- Clone the repo
  ```sh
  git clone https://github.com/goipay/goipay.git
  ```
- Build using [`make`](https://man7.org/linux/man-pages/man1/make.1.html)
  ```sh
  cd goipay && make build
  ```
- Under the `bin` folder you will find `server` binary
  ```sh
  ./bin/server -h
  
  Usage of ./bin/server:
    -client-ca string
          Comma-separated list of paths to client certificate authority files (for mTLS)
    -config string
          Path to the config file (default "config.yml")
    -log-level string
          Defines the logging level
    -reflection
          Enables gRPC server reflection
  ```
  
## Usage

- Get a quick overview of how GoiPay works by watching this [simple showcase video](https://youtu.be/b6TJBiHKJXE?feature=shared).
- Check out an [example project](https://github.com/goipay/example) to see GoiPay in action.
- For detailed information on using GoiPay's API, refer to the [API Reference](https://goipay.github.io/docs/api/grpc).

## Use cases

GoiPay is designed as a microservice that can be integrated into larger projects. If you need a simple, lightweight solution for just generating and processing crypto invoices, GoiPay is the perfect choice.
