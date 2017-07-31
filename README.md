# clustor [![Build Status](https://travis-ci.org/meijun/clustor.svg?branch=master)](https://travis-ci.org/meijun/clustor)
Simplest cluster monitor

### Build
- [Install Go](https://golang.org/doc/install)
- Download & build: `go get github.com/meijun/clustor`

### Deployment

The path of the executable binary is `$GOPATH/bin/clustor`,
and you can copy the binary to any host, without installing Go environment again.

- Run as a web server:
  - Copy the binary to the web server host.
  - Listen on a port:
    - `./clustor -listen :7160`
- Run as a worker:
  - Copy the binary to the worker host.
  - Send information to the web server:
    - `./clustor -send http://10.10.7.160:7160`
