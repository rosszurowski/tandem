build: dist/tandem

.github/demo.gif: demo/tandem.tape
	@cd demo && vhs tandem.tape

dist/tandem: go.mod go.sum $(shell fd -g '*.go' .)
	@go build -o dist/tandem .

release: .goreleaser.yml go.mod go.sum $(shell fd -g '*.go' .) ## Build and test release binaries
	@goreleaser release --snapshot --rm-dist
	@echo "Run 'git tag <tag-id>' and then 'git push --tags' to publish the release"

help: ## Show this help
	@echo "\nSpecify a command. The choices are:\n"
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[0;36m%-12s\033[m %s\n", $$1, $$2}'
	@echo ""
.PHONY: help
