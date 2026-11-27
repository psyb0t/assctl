DEV_IMAGE := assctl-dev
UID := $(shell id -u)
GID := $(shell id -g)
DEV_RUN := docker run --rm --init --cap-drop ALL --security-opt no-new-privileges:true \
	--user "$(UID):$(GID)" -e HOME=/tmp -e GOPATH=/tmp/go -e GOCACHE=/tmp/go-build \
	-v "$(CURDIR):/work" -w /work

.PHONY: all build check dep dev-image format lint lint-fix sec test test-unit test-integration test-coverage test-real test-fuzz help

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

lint-fix: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) golangci-lint run --fix

test: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) go test -mod=vendor -race -count=1 -timeout=5m ./...

test-unit: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) go test -mod=vendor -race -count=1 -timeout=5m -skip '^TestRender' ./...

test-integration: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) go test -mod=vendor -race -count=1 -timeout=5m -run '^TestRender' ./pkg/ass

test-real: dev-image
	$(DEV_RUN) -e ASSCTL_TEST_REAL_FONTS=1 $(DEV_IMAGE) go test -mod=vendor -race -count=1 -timeout=5m -v -tags real -run '^TestReal' ./internal/pkg/fonts

test-fuzz: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) sh -ec 'for name in FuzzParsePreservesBytes FuzzGeneratedOverrideRoundTrip; do go test -mod=vendor -race -run "^$$" -fuzz "^$$name$$" -fuzztime=10s -parallel=2 -timeout=2m ./pkg/ass; done'

test-coverage: dev-image
	$(DEV_RUN) --network none $(DEV_IMAGE) sh -ec 'go test -mod=vendor -race -count=1 -timeout=5m -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | awk "/^total:/{print \$$NF; if (\$$NF+0 < 90) exit 1}" > coverage-percent.txt && test -s coverage-percent.txt'

help:
	@printf '%s\n' 'build: compile the CLI' 'check: lint, complete tests, and build' 'format: format Go sources' 'lint / lint-fix: check or fix lint' 'test: offline unit and real renderer tests' 'test-unit: offline tests without renders' 'test-integration: FFmpeg/libass render tests' 'test-coverage: complete offline suite, minimum 90%' 'test-real: download and render every preset font' 'test-fuzz: bounded parser and generated-script fuzzing' 'sec: secrets and Go vulnerability scans' 'docs: build documentation' 'dep: refresh vendored dependencies'

sec: dev-image
	docker run --rm --network none --read-only --cap-drop ALL --security-opt no-new-privileges:true -v "$(CURDIR):/work:ro" ghcr.io/gitleaks/gitleaks@sha256:c00b6bd0aeb3071cbcb79009cb16a60dd9e0a7c60e2be9ab65d25e6bc8abbb7f dir --config /work/.gitleaks.toml --redact --no-banner /work
	$(DEV_RUN) $(DEV_IMAGE) sh -ec 'govulncheck -format sarif ./... > sec.sarif && govulncheck ./...'

check: lint test build

DOCS_NODE_IMAGE := node:24.21.0-bookworm-slim@sha256:d6aa754f16b3197301076f047b5def2f02ea1dbbc2ca920407d46d7ec7f87b20
DOCS_CACHE := .testing/docs-cache
DOCS_RUN := docker run --rm --init --cap-drop ALL --security-opt no-new-privileges:true \
	--user $(UID):$(GID) -e HOME=/tmp -e COREPACK_HOME=/cache/corepack \
	-v "$(CURDIR)/$(DOCS_CACHE):/cache" \
	-v "$(CURDIR)/docs:/work/docs:ro" -v "$(CURDIR)/website:/work/website" -w /work/website

.PHONY: docs docs-build docs-lock
docs: docs-build

docs-build: | $(DOCS_CACHE)
	$(DOCS_RUN) $(DOCS_NODE_IMAGE) corepack pnpm install --frozen-lockfile --ignore-scripts --store-dir /cache/pnpm
	$(DOCS_RUN) --network none $(DOCS_NODE_IMAGE) corepack pnpm run build

docs-lock: | $(DOCS_CACHE)
	$(DOCS_RUN) $(DOCS_NODE_IMAGE) corepack pnpm install --lockfile-only --ignore-scripts --store-dir /cache/pnpm

$(DOCS_CACHE):
	mkdir -p "$@"
