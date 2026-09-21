GOVERSION ?= 1.20
ROOT_DIR=${PWD}
HARDWARE=$(shell uname -m)
GIT_SHA=$(shell git --no-pager describe --always --dirty| cut -c1-7)
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
BUILD_TIME=$(shell date '+%s')
#VERSION ?= $(shell awk '/Release.*=/ { print $$3 }' version/version.go | sed 's/"//g')
ARCHITECTURES=amd64
HOST=$(shell hostname)
DOC_PACKAGE=github.com/paalgyula/summit/docs
LFLAGS ?= -X ${DOC_PACKAGE}.Gitsha=${GIT_SHA} \
	-X ${DOC_PACKAGE}.Version=prod \
    -X ${DOC_PACKAGE}.Compiled=${BUILD_TIME} \
    -X ${DOC_PACKAGE}.Buildhost=${HOST} \
    -X ${DOC_PACKAGE}.Branch=${GIT_BRANCH}

TAGS?=netgo

BUILDCMD=CGO_ENABLED=0 GOOS=linux GOARCH=${ARCHITECTURES} go build -a -tags ${TAGS} -ldflags "-s -w ${LFLAGS}" 

default: build

## Install dependencies required for code generating
deps:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
	go install golang.org/x/tools/cmd/stringer@latest
	go install github.com/josharian/impl@latest

## Generates interface stubs
gen: 
	@echo "Generating database interface stubs"

	@cd pkg/store/mysqldb && impl 'store *AccountStore' store.AccountRepo >> accountstore.go 
	@cd pkg/store/mysqldb && impl 'store *CharacterStore' store.CharacterRepo >> characterstore.go  
	@cd pkg/store/mysqldb && impl 'store *WorldStore' store.WorldRepo >> worldstore.go  

	@echo "Generating world server interface stubs"
	@cd pkg/summit/world && impl 'ws *Server' world.SessionManager >> sessionmanager.go

	@echo "Generating auth server interface stubs"
	@cd pkg/summit/auth && impl 'ms *ManagementServiceImpl' ManagementService >> management.go	
	@cd pkg/summit/auth && impl 'mc *ManagementClient' ManagementService >> management_client.go	


	@go install cmd/datagen/datagen.go
	@go generate ./...

clean:
	rm -Rf bin/*

lint:
	@echo "--> Linting the project with golangci-lint..."
	@golangci-lint run ./...

build:
	@echo "--> Compiling the project"
	@mkdir -p bin/
	go build -o bin/summit cmd/summit/summit.go
	go build -o bin/serworm cmd/serworm/serworm.go
	go build -o bin/datagen cmd/datagen/datagen.go
	go build -o bin/assetserver cmd/assetserver/main.go
	go build -o bin/summitctl cmd/summitctl/summitctl.go

## Run the whole local dev stack (game server, asset server, web client) with restart on rebuild
dev:
	@scripts/dev.sh

## Build, push and roll out the asset server to Kubernetes, evicting its cache
deploy-assetserver:
	@scripts/deploy-assetserver.sh

## Build, push and roll out the game server and web client to Kubernetes
deploy-summit:
	@scripts/deploy-summit.sh

## Wipe the asset server's conversion cache in the running pod
evict-asset-cache:
	@scripts/deploy-assetserver.sh evict-cache

build-dist: clean
	@mkdir -p bin/
	@echo "--> Compiling world server"
	@$(BUILDCMD) -o bin/summit cmd/summit/summit.go
	@echo "--> Compiling serworm"
	@$(BUILDCMD) -o bin/serworm cmd/serworm/serworm.go
	@echo "--> Compiling datagen"
	@$(BUILDCMD) -o bin/datagen cmd/datagen/datagen.go
	@echo "--> Compiling assetserver"
	@$(BUILDCMD) -o bin/assetserver cmd/assetserver/main.go
	@echo "--> Compiling summitctl"
	@$(BUILDCMD) -o bin/summitctl cmd/summitctl/summitctl.go
	@echo "--> Compressing binaries with UPX for small bandwidth environment..."
	@command -v upx >/dev/null 2>&1 && upx -9 bin/* || echo "UPX not available, skipping"
	@echo "Done. You can find the compiled binaries in the bin/ folder"

## Compress compiled binaries in bin/ with UPX
upx:
	@echo "--> Compressing binaries in bin/ with UPX..."
	@command -v upx >/dev/null 2>&1 && upx -9 bin/* || (echo "UPX is not installed" && exit 1)

## Installs dependencies (summit code generation tools) to go's bin folder. Usually to $HOME/go/bin
install:	
	@echo "Installing summit tools..."
	@go install cmd/datagen/datagen.go
	@go install cmd/summitctl/summitctl.go
