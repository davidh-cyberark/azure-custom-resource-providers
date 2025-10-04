# Makefile
# Time-stamp: <Sat Oct 4 00:53:40 UTC 2025>

.PHONY: all
all:
	echo "noop"

.PHONY: rebuild-and-deploy
rebuild-and-deploy:
	(cd custom-provider && make versionbump && make vardump)
	./build-docker.sh --no-test --push 
	./deploy-infrastructure.sh -y --debug
	./update-containerapp.sh

.PHONY: fetch-container-logs
fetch-container-logs:
	./fetch-container-logs.sh
