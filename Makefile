.PHONY: run generate serve build-web install

run: generate serve

generate:
	go run ./cmd/cortexdocs generate ./examples/sample-c-api

serve:
	go run ./cmd/cortexdocs serve

build-web:
	cd web && npm install && npm run build

install:
	go install ./cmd/cortexdocs
