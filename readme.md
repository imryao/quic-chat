# quic-chat

A simple chat that works over [QUIC](https://en.wikipedia.org/wiki/QUIC).

## Running Server

1. Generate a set of private and public keys:

```shell
openssl genpkey -algorithm ed25519 -out server.key
openssl req -new -x509 -key server.key -out server.crt -days 3650
```

2. Run the server:

```shell
go run ./cmd/server
```

## Running Client

Run the client:

```shell
go run ./cmd/client [-s <ServerAddress>] -n <Nickname>
```
