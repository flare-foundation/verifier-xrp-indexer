#!/bin/bash

go test -v -coverpkg=./... -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
