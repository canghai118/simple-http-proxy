
build:
	go build

container:
	GOOS=linux go build
	docker build -t canghai/simple-http-proxy .