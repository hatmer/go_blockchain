# Go Blockchain
![goreportcard](https://goreportcard.com/badge/github.com/hatmer/go_blockchain)

Simple blockchain implementation in Go. 

## What it does
* Logs url of requests to webserver as a PoW-secured blockchain on disk.

## Design Goals
* Simple and elegant solution.
* Proper Go code architecture.
* 100% Test Coverage.

## Notes
* Runs out of the box using `go run main.go`. No dependencies.
* Is not fully tested for production use.
* Difficulty setting > 2 doesn't work due to a bug in the sha256 algorithm.

