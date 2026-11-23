.PHONY: test run smoke coverage

test:
	go test ./...
	npm test

run:
	go run ./cmd/server

smoke:
	bash scripts/smoke.sh

coverage:
	TEST_DATABASE_URL=$${TEST_DATABASE_URL:-postgresql:///unwind_test?host=/tmp} go test -coverprofile=/tmp/unwind-coverage.out ./...
	go tool cover -func=/tmp/unwind-coverage.out
