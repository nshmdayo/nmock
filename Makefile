start:
	docker build -t nmock-server ./app
	docker run --name nmock-server -p 9000:9000 -v $(PWD)/app/config.json:/app/config.json -it nmock-server

stop:
	docker stop nmock-server

clean:
	docker rm nmock-server
	docker rmi nmock-server

restart: stop start

# Local development commands
dev:
	cd app && go run main.go

build:
	cd app && go build -o ../output/nmock main.go
