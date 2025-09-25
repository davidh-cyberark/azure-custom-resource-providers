# Makefile

.PHONY: Makefile

BINDIR := bin
APPNAME := main
VERSION_START := 0.0.1

GO := $(shell command -v go 2> /dev/null)
VER := $(shell [ -f VERSION ] && cat VERSION || printf $(VERSION_START))
DATE_TIME := $(shell date +"%Y-%m-%d %H:%M:%S")
LDFLAGS := -ldflags "-X 'main.Version=$(VER)' -X 'main.BuildDate=$(DATE_TIME)'"

.PHONY: build
build: $(APPNAME)

# Build the main executable with the default GOOS and GOARCH
$(APPNAME): VERSION main.go
	 CGO_ENABLED=0 GOOS=linux $(GO) build -a -installsuffix cgo -o $@ $(LDFLAGS)

VERSION:
	printf $(VERSION_START) > VERSION

.PHONY: versionbump
versionbump:
	@awk -F. '{printf "%s.%s.%d\n", $$1, $$2, $$3+1 > "VERSION"}' VERSION

.PHONY: clean vardump
clean:
	rm -f $(APPNAME)

vardump:
	@echo "=== Makefile Variables ==="
	@echo "BINDIR      = $(BINDIR)"
	@echo "APPNAME     = $(APPNAME)"
	@echo "GO          = $(GO)"
	@echo "VERSION     = $(VER)"
	@echo "DATE_TIME   = $(DATE_TIME)"
	@echo "LDFLAGS     = $(LDFLAGS)"
	@echo "=========================="
