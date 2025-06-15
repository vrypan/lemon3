SNAPCHAIN_VER := $(shell cat SNAPCHAIN_VERSION 2>/dev/null || echo "unset")
LEMON3_VERSION := $(shell git describe --tags 2>/dev/null || echo "v0.0.0")

BINS = lemon3
PROTO_FILES := $(wildcard proto/*.proto)
LEMON3_SOURCES := $(wildcard */*.go go.mod)

# Colors for output
GREEN = \033[0;32m
NC = \033[0m

all: $(BINS)

clean:
	@echo -e "$(GREEN)Deleting lemon3 binary...$(NC)"
	rm -f $(BINS)

.PHONY: all clean local release-notes tag tag-minor tag-major releases

lemon3: $(LEMON3_SOURCES)
	@echo -e "$(GREEN)Building lemon3 ${LEMON3_VERSION} $(NC)"
	go build -o $@ -ldflags "-w -s -X main.LEMON3_VERSION=${LEMON3_VERSION}"

release-notes:
	# Automatically generate release_notes.md
	./bin/generate_release_notes.sh

tag:
	./bin/auto_increment_tag.sh patch

tag-minor:
	./bin/auto_increment_tag.sh minor

tag-major:
	./bin/auto_increment_tag.sh major

releases:
	goreleaser release --clean
