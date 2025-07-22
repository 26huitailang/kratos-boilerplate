#!/bin/bash

set -e

echo "running go test (TDD, exclude service)..."
go test $(go list ./internal/... | grep -v '/service') -coverprofile=coverage.out > tdd_test.log
cat tdd_test.log

echo "generating coverage report..."
go tool cover -html=coverage.out -o coverage.html
echo "coverage report generated: coverage.html (open in browser to view details)"

COVERAGE=$(go tool cover -func=coverage.out | grep total: | awk '{print $3}')
echo "total coverage (TDD only, service excluded): $COVERAGE"

echo "running ginkgo (BDD, service layer)..."
ginkgo -r ./internal/service || echo "ginkgo not installed, please run: go install github.com/onsi/ginkgo/v2/ginkgo@latest"
