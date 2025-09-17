# Makefile

.PHONY: Makefile

BINDIR := bin
APPNAME := main

GO := $(shell command -v go 2> /dev/null)
VER := $(shell [ -f VERSION ] && cat VERSION || printf "0.0.1")
DATE_TIME := $(shell date +"%Y-%m-%d %H:%M:%S")
LDFLAGS := -ldflags "-X 'main.Version=$(VER)' -X 'main.BuildDate=$(DATE_TIME)'"

.PHONY: build
build: $(APPNAME)

# Build the main executable with the default GOOS and GOARCH
$(APPNAME): VERSION main.go
	# 	$(GO) build -o $@ $(LDFLAGS)
	#RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
	CGO_ENABLED=0 GOOS=linux $(GO) build -a -installsuffix cgo -o $@ $(LDFLAGS)

VERSION:
	printf "0.0.1" > VERSION

.PHONY: clean vardump
clean:
	rm -f $(APPNAME)

vardump:
	@echo "=== Makefile Variables ==="
	@echo "BINDIR      = $(BINDIR)"
	@echo "APPNAME     = $(APPNAME)"
	@echo "GO          = $(GO)"
	@echo "VER         = $(VER)"
	@echo "DATE_TIME   = $(DATE_TIME)"
	@echo "LDFLAGS     = $(LDFLAGS)"
	@echo "=========================="
