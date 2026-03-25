.PHONY: build install test vet check skill-sync all

all: check install skill-sync

build:
	go build ./...

test:
	go test ./... -count=1

vet:
	go vet ./...

check: build test vet

install:
	go build -o /usr/local/bin/goalx ./cmd/goalx

skill-sync:
	@mkdir -p ~/.claude/skills/goalx/references
	cp skill/SKILL.md ~/.claude/skills/goalx/SKILL.md
	cp skill/references/advanced-control.md ~/.claude/skills/goalx/references/advanced-control.md
	@echo "✓ skill synced to ~/.claude/skills/goalx/"
