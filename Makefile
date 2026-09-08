.PHONY: tidy test race up down logs fmt smoke

tidy:
	go mod tidy

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './.git/*')

test:
	go test ./...

race:
	go test -race ./...

up:
	docker compose up --build -d

down:
	docker compose down

logs:
	docker compose logs -f

smoke:
	bash scripts/smoke.sh
