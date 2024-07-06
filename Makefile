.PHONY: clean build start

clean:
	@go clean
	@rm -rf ./bin

build: clean
	# docker run --rm -v "$(shell pwd)/services/user-service":/usr/src/app -w /usr/src/app golang:alpine go build -o /usr/src/app/bin/user-service main.go
	# docker run --rm -v "$(shell pwd)/services/event-service":/usr/src/app -w /usr/src/app golang:alpine go build -o /usr/src/app/bin/event-service main.go
	# docker run --rm -v "$(shell pwd)/services/registration-service":/usr/src/app -w /usr/src/app golang:alpine go build -o /usr/src/app/bin/registration-service main.go
	cd services/user-service && go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../../bin/user-service
	cd services/event-service && go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../../bin/event-service
	cd services/registration-service && go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../../bin/registration-service

start:
	docker compose up --build

