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

BUILDCMD=CGO_ENABLED=0 GOOS=linux go build -a -tags ${TAGS} -ldflags "-s -w ${LFLAGS}" 

default: build

## Install dependencies required for code generating
deps:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0
	go install golang.org/x/tools/cmd/stringer@latest
	go install github.com/josharian/impl@latest

## Generates interface stubs
gen: 
	@echo "Generating database interface stubs"
	@cd pkg/store/localdb && impl 'repo *LocalStore' store.AccountRepository >> localstore.go 
	@cd pkg/store/localdb && impl 'repo *LocalStore' store.CharacterRepository >> localstore.go  
	@cd pkg/store/localdb && impl 'repo *LocalStore' store.WorldRepository >> localstore.go  

	@cd pkg/store/mysqldb && impl 'store *AccountStore' store.AccountRepository >> accountstore.go 
	@cd pkg/store/mysqldb && impl 'store *CharacterStore' store.CharacterRepository >> characterstore.go  
	@cd pkg/store/mysqldb && impl 'store *WorldStore' store.WorldRepository >> worldstore.go  

	@echo "Generating world server interface stubs"
	@cd pkg/summit/world && impl 'ws *Server' world.SessionManager >> sessionmanager.go

	@echo "Generating auth server interface stubs"
	@cd pkg/summit/auth && impl 'ms *ManagementServiceImpl' ManagementService >> management.go	

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

build-dist: clean
	@mkdir -p bin/
	@echo "--> Compiling world server"
	@$(BUILDCMD) -o bin/summit cmd/summit/summit.go
	@echo "--> Compiling serworm"
	@$(BUILDCMD) -o bin/serworm cmd/serworm/serworm.go
	@echo "--> Compiling datagen"
	@$(BUILDCMD) -o bin/datagen cmd/datagen/datagen.go
	@echo "Done. You can find the compiled binaries in the bin/ folder"

## Installs dependencies (summit code generation tools) to go's bin folder. Usually to $HOME/go/bin
install:	
	@echo "Installing summit tools..."
	@go install cmd/datagen/datagen.go
