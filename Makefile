.PHONY: up down build test fmt

up:
	docker compose up -d

down:
	docker compose down

build: build-backend build-console

build-backend:
	cd backend && go build ./...

build-console:
	melos build:web

test: test-backend test-flutter test-sdk-js

test-backend:
	cd backend && go test ./...

test-flutter:
	melos test

test-sdk-js:
	cd sdks/js && npm test

fmt:
	cd backend && gofmt -w .
	melos format

bootstrap:
	dart pub global activate melos
	melos bootstrap
	cd sdks/js && npm install

migrate:
	cd backend && go run ./cmd/migrate
