HUBBLE_VER := 1.19.1
LEMON3_VER := $(shell git describe --tags 2>/dev/null || echo "v0.0.0")

BINS = lemon3
PROTO_FILES := $(wildcard schemas/*.proto)
SOURCES := $(wildcard utils/*.go cmd/*.go config/*.go enclosure/*.go fctools/*.go ipfsServer/*.go ui/*.go)

# Colors for output
GREEN = \033[0;32m
NC = \033[0m

all: $(BINS)

# Build binaries, depends on compiled protos
# $(BINS): .farcaster-built
#	@echo -e "$(GREEN)Building $@...$(NC)"
#	go build -o $@ ./cmd/$@

# Compile .proto files, touch stamp file
.farcaster-built: $(PROTO_FILES)
	@echo -e "$(GREEN)Compiling .proto files...$(NC)"
	protoc --proto_path=schemas --go_out=. \
	$(shell cd schemas; ls | xargs -I \{\} echo -n '--go_opt=M'{}=farcaster/" " '--go-grpc_opt=M'{}=farcaster/" " ) \
	--go-grpc_out=. \
	schemas/*.proto
	@touch .farcaster-built

proto:
	@echo -e "$(GREEN)Downloading proto files (Hubble v$(HUBBLE_VER))...$(NC)"
	curl -s -L "https://github.com/farcasterxyz/hub-monorepo/archive/refs/tags/@farcaster/hubble@$(HUBBLE_VER).tar.gz" \
	| tar -zxvf - -C . --strip-components 2 "hub-monorepo--farcaster-hubble-$(HUBBLE_VER)/protobufs/schemas/"
	sed -i '' 's|syntax = "proto3";|syntax = "proto3"; \
	option go_package = "github.com/vrypan/lemon3/farcaster/";|' schemas/*.proto

clean:
	@echo -e "$(GREEN)Cleaning up...$(NC)"
	rm -f $(BINS) farcaster/*.pb.go farcaster/*.pb.gw.go .farcaster-built

.PHONY: all proto clean local release-notes tag tag-minor tag-major releases

lemon3: .farcaster-built $(SOURCES)
	@echo -e "$(GREEN)Building lemon3 ${LEMON3_VER} $(NC)"
	go build -o $@ -ldflags "-w -s -X github.com/vrypan/lemon3/config.LEMON3_VERSION=${LEMON3_VER} \
	-X google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=ignore"


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
