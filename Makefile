build:
	cd client && CGO_ENABLED=0 go build
	cd deviceconnect && CGO_ENABLED=0 go build
	docker-compose build 
up:
	docker-compose up

