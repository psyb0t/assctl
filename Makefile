DEV_IMAGE := assctl-dev
UID := $(shell id -u)
GID := $(shell id -g)
DEV_RUN := docker run --rm --init --cap-drop ALL --security-opt no-new-privileges:true \
	--user "$(UID):$(GID)" -e HOME=/tmp -e GOPATH=/tmp/go -e GOCACHE=/tmp/go-build \
	-v "$(CURDIR):/work" -w /work

.PHONY: all build check dep dev-image format lint sec test test-coverage

all: check

dev-image:
	docker build -f Dockerfile.dev -t $(DEV_IMAGE) .

dep: dev-image
	$(DEV_RUN) $(DEV_IMAGE) sh -ec 'go mod tidy && go mod vendor'

format: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) golangci-lint fmt

build: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) go build -mod=vendor -trimpath -o bin/assctl ./cmd/assctl

lint: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) golangci-lint run

test: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) go test -mod=vendor -race -count=1 ./...

test-coverage: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) sh -ec 'go test -mod=vendor -race -count=1 -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | awk "/^total:/{print \$$NF}" > coverage-percent.txt && test -s coverage-percent.txt'

sec: dev-image
	docker run --rm --network none --read-only --cap-drop ALL --security-opt no-new-privileges:true -v "$(CURDIR):/work:ro" ghcr.io/gitleaks/gitleaks@sha256:c00b6bd0aeb3071cbcb79009cb16a60dd9e0a7c60e2be9ab65d25e6bc8abbb7f dir --config /work/.gitleaks.toml --redact --no-banner /work
	$(DEV_RUN) $(DEV_IMAGE) sh -ec 'govulncheck -format sarif ./... > sec.sarif && govulncheck ./...'

check: lint test build
