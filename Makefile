docker-up:
	docker-compose up -d --build

docker-down:
	docker-compose down -v

test: 
	go test -timeout=2m ./...