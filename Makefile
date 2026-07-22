.PHONY: up down build test fmt

up:
	docker compose up -d --build

down:
	docker compose down

build: build-backend build-console

build-backend:
	cd apps/backend && go build ./...

build-console:
	cd apps/console && npm run build

test: test-backend test-flutter test-sdk-js

test-backend:
	cd apps/backend && go test ./...

test-flutter:
	melos test

test-sdk-js:
	cd sdks/js && npm test

fmt:
	cd apps/backend && gofmt -w .
	melos format

bootstrap:
	dart pub global activate melos
	melos bootstrap
	cd sdks/js && npm install

migrate:
	cd apps/backend && go run ./cmd/migrate
