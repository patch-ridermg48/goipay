# GoiPay

<div align="center">
<img src="https://raw.githubusercontent.com/goipay/goipay.github.io/refs/heads/master/static/img/goipay-logo-with-name.svg" alt="Logo" width="400" height="100">
</div>

## Description
> **Note:**
> The project is in development. This is not a release version.  
> As for now, only XMR invoices are implemented.

A lightweight crypto payment processor microservice, written in Golang, designed for creating and processing cryptocurrency invoices via gRPC.

## Getting Started
### Prerequisites
- Go ≥ 1.22
- PostgreSQL ≥ 12

### Installation
#### Docker
- Clone the repo
  ```sh
  https://github.com/chekist32/goipay.git
  ```
- Inside the root dir create and populate ```.env``` file on the base of ```.env.example``` file
  ```ini
  # Can be either 'prod' or 'dev'.
  # In 'dev' mode, a reflection server is established.
  MODE=dev

  SERVER_HOST=localhost
  SERVER_PORT=3000

  # As for now, only PostgreSQL is supported
  DATABASE_HOST=localhost
  DATABASE_PORT=54321
  DATABASE_USER=postgres
  DATABASE_PASS=postgres
  DATABASE_NAME=goipay_db

  XMR_DAEMON_URL=http://node.monerodevs.org:38089
  XMR_DAEMON_USER=
  XMR_DAEMON_PASS=
  ```
- Inside the root dir you can find an example ```docker-compose.yml``` file. For testing purposes can be run without editing.
  ```sh
  docker compose up
  ```
