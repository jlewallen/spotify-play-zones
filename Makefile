all: build

build: *.go tokens.json
	go build -o play-zones *.go
	docker build -t jlewallen/play-zones:master .

push: build
	docker push jlewallen/play-zones:master
