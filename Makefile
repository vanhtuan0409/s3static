.PHONY: build
VERSION=0.1.0-rel1
IMAGE=gregistry.vanhtuan0409.com/vanhtuan0409/s3static

clean:
	rm -rf ./bin

build:
	CGO_ENABLED=0 go build -o bin/s3static .

build-docker: build
	docker build -t ${IMAGE}:${VERSION} .

publish-docker: build-docker
	docker push ${IMAGE}:${VERSION}
