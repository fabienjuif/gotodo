build:
	@cd services/todos; GOARCH=amd64 GOOS=linux go build .

start: build
	@docker compose up --remove-orphans

clean:
	@rm services/todos/todos
	@docker compose down -v

curl:
	@curl -XPOST "http://localhost:8080/2015-03-31/functions/function/invocations" -d '{}'