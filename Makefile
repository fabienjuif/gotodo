.PHONY: build stop start local-get

build:
	sam build

stop:
	docker compose down -v

start:
	docker compose up -d

local-get:
	sam local invoke --region eu-west-3 --docker-network local-dynamodb -e events/get.json 

local-post:
	sam local invoke --region eu-west-3 --docker-network local-dynamodb -e events/post.json