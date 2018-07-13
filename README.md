# Mapnificent Generator - Beta

This repo contains a Beta version of the Mapnificent Generator in Go. It takes GTFS feeds and converts them to Protocol Buffers that can be read by mapnificent.net.

## Setting up the environment

### Installing go dependencies

	# You need go (do not forget to set $GOPATH)
	go get github.com/golang/protobuf/proto
	go get github.com/mapnificent/gogtfs
	go get github.com/mapnificent/mapnificent_generator/mapnificent.pb


### Installing python dependencies

	# You need pipenv
	pipenv --three install


### Build distribution of the Mapnificent Generator

	sh dist.sh


### Setting transitfeed API key

	export TRANSITFEED_API_KEY=<Your transitfeeds.com API Key>


## Tasks

### Download and update city automatically

	# for example: pipenv run python -m scripts.download ~/mapnificent/_cities/aachen
	pipenv run python -m scripts.download <mapnificent city directory containing markdown file>


### Create a new city

	# Additional information (cityid, cityname, coords, ...) are queried by the script on execution
	# for example: pipenv run python -m scripts.create ~/mapnificent/_cities
	pipenv run python -m scripts.create <mapnificent cities directory>


### Create directly from GTFS

	# for example: go run mapnificent.go -d ~/bolzano.zip -o ~/bolzano.bin -v
	go run mapnificent.go -d <dir of GTFS files> -o <outputfile> -v


### Compile Protocol Buffer Definition to Go file

    protoc -I=mapnificent.pb --go_out=mapnificent.pb mapnificent.pb/mapnificent.proto

