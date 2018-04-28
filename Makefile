BUILD=go build
CLEAN=go clean
INSTALL=go install
BUILDPATH=./_build
PACKAGES=$(shell go list ./... |grep -v vendor/)
EXAMPLES=$(shell find examples/* -maxdepth 0 -type d -exec basename {} \;)

examples: builddir
	for example in $(EXAMPLES); do \
		go build -o "$(BUILDPATH)/$$example" "examples/$$example/$$example.go"; \
	done

all: examples

slackertify: builddir
	go build -o "$(BUILDPATH)/slackertify" "examples/slackertify/slackertify.go"

builddir:
	mkdir -p $(BUILDPATH)

install:
	$(INSTALL) ./$(EXDIR)/...

clean:
	rm -rf $(BUILDPATH)
	$(CLEAN)

godep:
	go get -u github.com/golang/dep/cmd/dep

dep: godep
	dep ensure

check:
	for pkg in ${PACKAGES}; do \
		go vet -composites=false $$pkg || exit ; \
		golint $$pkg || exit ; \
	done

test:
	for pkg in ${PACKAGES}; do \
		go test -coverprofile="../../../$$pkg/coverage.txt" -covermode=atomic $$pkg || exit; \
	done

.PHONY: clean examples
