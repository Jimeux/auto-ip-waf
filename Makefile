.PHONY: build clean deploy

build:
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/api api/main.go
	env GOARCH=amd64 GOOS=linux go build -ldflags="-s -w" -o bin/iprefresher iprefresher/main.go

clean:
	rm -rf ./bin ./vendor

deploy: clean build
	sls deploy --verbose

deploy-api: clean build
	sls deploy -f api

deploy-ip: clean build
	sls deploy -f iprefresher
