.PHONY: help
help:			## Show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

.PHONY: vendor
vendor:			## Populate the vendor directory
		go mod vendor

VERSION=$$(git tag -l|sort -t. -k 1,1r -k 2,2nr -k 3,3nr -k 4,4nr|head -n1)
GIT_COMMIT=$$(git rev-parse --verify HEAD)

.PHONY: build
build: vendor		## Build the binary
		@CGO_ENABLED=0 go build -mod=vendor -a -ldflags="-X github.com/lossanarch/dockfmt/version.VERSION=$(VERSION) -X github.com/lossanarch/dockfmt/version.GITCOMMIT=$(GIT_COMMIT)"

.PHONY: install
install:		## Install the binary (via go install)
		@CGO_ENABLED=0 go install -mod=vendor -a -ldflags="-w -s -X github.com/lossanarch/dockfmt/version.VERSION=$(VERSION) -X github.com/lossanarch/dockfmt/version.GITCOMMIT=$(GIT_COMMIT)"

.PHONY: docker
docker: vendor		## Build the docker image
		@docker build -t dockfmt:latest .

.PHONY: deploy
deploy: docker
		docker tag dockfmt:latest lossanarch/dockfmt:$(VERSION)
		docker tag dockfmt:latest lossanarch/dockfmt:latest
		docker push lossanarch/dockfmt:$(VERSION)
		docker push lossanarch/dockfmt:latest