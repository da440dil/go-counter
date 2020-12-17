# go-counter

[![Build Status](https://travis-ci.com/da440dil/go-counter.svg?branch=master)](https://travis-ci.com/da440dil/go-counter)
[![Coverage Status](https://coveralls.io/repos/github/da440dil/go-counter/badge.svg?branch=master)](https://coveralls.io/github/da440dil/go-counter?branch=master)
[![GoDoc](https://godoc.org/github.com/da440dil/go-counter?status.svg)](https://godoc.org/github.com/da440dil/go-counter)
[![Go Report Card](https://goreportcard.com/badge/github.com/da440dil/go-counter)](https://goreportcard.com/report/github.com/da440dil/go-counter)

Distributed rate limiting using [Redis](https://redis.io/).

Example usage:

- [example](./examples/fixedwindow/main.go) using [fixed window](./fixedwindow.go) algorithm 

    ```go run examples/fixedwindow/main.go```
- [example](./examples/slidingwindow/main.go) using [sliding window](./slidingwindow.go) algorithm

    ```go run examples/slidingwindow/main.go```
