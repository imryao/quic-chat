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
go run ./cmd/server -n 127.0.0.1:9080 -q 4242 -h 9080 -b 16 -c server.crt -k server.key
```

## Running Client

Run the client:

```shell
go run ./cmd/client -n mryao-client-1 -s 127.0.0.1:4242 -b 16
go run ./cmd/client -n mryao-client-2 -s 127.0.0.1:4242 -b 16
go run ./cmd/client -n mryao-client-3 -s 127.0.0.1:4242 -b 16
```
