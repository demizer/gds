#!/bin/bash

GOPATH=$PWD:$PWD/vendor

go test -cover core -coverprofile=coverage.out
go tool cover -html=coverage.out
