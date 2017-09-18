# Mapnificent Generator - Beta

This repo contains a Beta version of the Mapnificent Generator in Go. It takes GTFS feeds and converts them to Protocol Buffers that can be read by mapnificent.net.

## Build distribution of the Mapnificent Generator

	sh dist.sh

## Download and update Mapnificent City automatically

	export TRANSITFEED_API_KEY=<Your transitfeeds.com API Key>
	python -m scripts.download <path to mapnificent city directory containing markdown file>

## Create a new city

	python -m scripts.create <path to mapnificent cities directory with all cities>


## Development tasks

### Run Generator directly on GTFS files in a directory

	go run mapnificent.go -h
	go run mapnificent.go -d <dir of GTFS files> -o <outputfile> -v

	# Run with output containing extra debug info
	go run mapnificent.go -d <dir of GTFS files> -o <outputfile> -v -e


### Compile Protocol Buffer Definition to Go file

    protoc -I=mapnificent.pb --go_out=mapnificent.pb mapnificent.pb/mapnificent.proto