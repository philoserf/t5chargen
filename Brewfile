# Brewfile — development toolchain for t5chargen.
# Install with `task deps` (or `brew bundle`).

brew "go"             # Go compiler/toolchain (see go.mod for required version)
brew "go-task"        # `task` runner — this project's build gate
brew "golangci-lint"  # meta-linter; enforced via `task lint`
brew "prettier"       # JSON and Markdown formatter; enforced via `task fmt:check:docs`

# nilaway has no formula; `task nilaway` needs:
#   go install go.uber.org/nilaway/cmd/nilaway@latest
