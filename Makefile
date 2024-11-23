start:
	docker build -t sample-golnag-server ./app
	docker run --name sample-golnag-server -p 9000:9000 -it sample-golnag-server

stop:
	docker stop sample-golnag-server
	docker rm sample-golnag-server
	docker rmi sample-golnag-server

restart: stop start