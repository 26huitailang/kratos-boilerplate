GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "find internal -name *.proto")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find api -name *.proto")
else
	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find api -name *.proto)
endif

.PHONY: init
# init env
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest

.PHONY: config
# generate internal proto
config:
	protoc --proto_path=./internal \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./internal \
	       $(INTERNAL_PROTO_FILES)

.PHONY: api
# generate api proto
api:
	protoc --proto_path=./api \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./api \
 	       --go-http_out=paths=source_relative:./api \
 	       --go-grpc_out=paths=source_relative:./api \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)
	@echo "API proto files generated successfully"
	@echo "OpenAPI documentation generated in current directory"
	@ls -la *.yaml 2>/dev/null | grep -E "(openapi|swagger)" || echo "Checking for generated OpenAPI files..."

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: generate
# generate
generate:
	go generate ./...
	go mod tidy

.PHONY: all
# generate all
all:
	make api;
	make config;
	make generate;

.PHONY: test
# run all tests
test:
	go test -v -cover ./internal/...

.PHONY: test-coverage
# run tests and generate coverage report
test-coverage:
	# 仅统计TDD覆盖率，排除internal/service目录
	go test $(go list ./internal/... | grep -v '/service') -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "coverage report generated: coverage.html (open in browser to view details)"
	@go tool cover -func=coverage.out | grep total:

.PHONY: test-coverage-html
# run tests and generate html coverage report
test-coverage-html: test-coverage
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test-clean
# clean test generated files
test-clean:
	rm -f coverage.out coverage.html

.PHONY: test-all
# regenerate all code and run tests
test-all:
	make api
	make generate
	make test

.PHONY: logcheck
# check log usage compliance
logcheck:
	@echo "Building log checker..."
	@cd tools/logchecker && go build -o logchecker .
	@echo "Running log compliance check..."
	@cd tools/logchecker && ./logchecker -dir ../../internal -config logchecker.json

.PHONY: logcheck-json
# check log usage compliance and output JSON report
logcheck-json:
	@cd tools/logchecker && go build -o logchecker .
	@cd tools/logchecker && ./logchecker -dir ../../internal -config logchecker.json -output json

.PHONY: logcheck-html
# check log usage compliance and output HTML report
logcheck-html:
	@cd tools/logchecker && go build -o logchecker .
	@cd tools/logchecker && ./logchecker -dir ../../internal -config logchecker.json -output html > ../../logcheck-report.html
	@echo "HTML report generated: logcheck-report.html"

.PHONY: logcheck-install
# install log checker tool
logcheck-install:
	@cd tools/logchecker && go build -o $(GOPATH)/bin/logchecker .
	@echo "Log checker installed to $(GOPATH)/bin/logchecker"

.PHONY: archive
# create source code archive
archive:
	@echo "Creating source code archive..."
	git archive --format=tar.gz --prefix=kratos-boilerplate-$(VERSION)/ HEAD -o kratos-boilerplate-$(VERSION).tar.gz
	@echo "Archive created: kratos-boilerplate-$(VERSION).tar.gz"

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
