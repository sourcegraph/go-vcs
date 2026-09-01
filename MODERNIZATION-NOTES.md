# Modernization notes

Judgment calls made while converting this repository to Go modules (Go 1.26).

## Dropped package

* **`vcs/util/tracer` was removed.** It existed only to wrap repositories with
  [appdash](https://github.com/sourcegraph/appdash) tracing. `appdash` cannot be
  built under its live path at *any* version: its `go.mod` declares
  `github.com/sourcegraph/appdash`, but its own sources still import
  `sourcegraph.com/sourcegraph/appdash/internal/wire`, so the build fails with
  "use of internal package ... not allowed". A `replace` back to the vanity path
  fails too ("used for two different module paths"), and the package's API is
  `*appdash.Recorder`, so there is nothing small to inline. Keeping the dead
  `sourcegraph.com/...` import was the only alternative, and rule 3 rules it out.
  The package had no tests; no other package in this repo imported it.

## Dependencies pinned to old / deprecated versions

* **`github.com/sqs/pbtypes v0.0.0-20180604144634-d3ebe8f20ae4`** — the tagged
  `v1.0.0` and current master both declare `module sourcegraph.com/sqs/pbtypes`,
  so they cannot be required under the live GitHub path. The pinned commit is the
  last one before `go.mod` was added, so its module path is inferred from the
  import path. `pbtypes.Timestamp` is part of this repo's public API
  (`vcs.Signature.Date`), so inlining it under `internal/` would have been a
  breaking API change. Side effect: `go mod tidy` records
  `sourcegraph.com/sqs/pbtypes v1.0.0` as an indirect requirement, because an
  external test file in the pinned commit imports the package by its old path.
* **`golang.org/x/tools/godoc v0.1.0-deprecated`** — `godoc/vfs` was deleted from
  `golang.org/x/tools` after v0.36.0 and republished as this explicitly
  deprecated module. `vfs.FileSystem` is the return type of
  `vcs.Repository.FileSystem`, so swapping it for `io/fs` would be a breaking API
  change. The alternative, pinning all of `golang.org/x/tools` to v0.36.0, drags
  in a much larger dependency for the same code.
* **`github.com/gogo/protobuf v1.3.2`** — archived and deprecated upstream, but
  `vcs/vcs.pb.go` is gogo-generated and uses gogo custom types and marshalers.
  Regenerating with `google.golang.org/protobuf` would change the public API.
* **`github.com/sourcegraph/go-git` (2019), `github.com/beyang/hgo` (2015),
  `github.com/flynn/go-shlex` (2015)** — unmaintained, pinned at pseudo-versions.
  They still build and have no API-compatible successors.

## Tests

* **The git-over-SSH tests skip at runtime** when `ssh` or `git-shell` is not on
  `$PATH` (`requireGitSSH` in `vcs/ssh_test.go`). They pass when both are
  installed, which is the case on `ubuntu-latest`.
* **The Mercurial test cases stay disabled**, as they already were before this
  change (issue #104). The three `hg` test helpers used an early `return`
  followed by the real setup code, which `go vet` rejects as unreachable; the
  early return is now behind `hgTestsDisabled()`, so the setup code is preserved
  verbatim and can be re-enabled by flipping one function. `hg` is also not
  installed in CI.
* **`TestRepository_FileSystem_gitSubmodules` now runs `git -c
  protocol.file.allow=always submodule add`**, because git 2.38+ refuses the
  `file` transport for submodules (CVE-2022-39253).

## Other

* **Unreachable fallback code in `vcs/git/repo.go` was deleted** (in
  `ResolveTag`, `Branches`, `Tags` and `Commits`). Each of those methods returns
  the `gitcmd` fallback on the first line and was followed by a never-executed
  native implementation that `go vet` flags. The `TODO`s explaining that the
  native implementation is still missing were kept; the deleted code remains in
  git history.
* **`vcs/gen.go` and the imports in `vcs.proto` were rewritten** to resolve
  include directories with `go list -m` instead of `$GOPATH/src`. This could not
  be executed here (neither `protoc` nor `protoc-gen-gogo` is installed), so the
  directive itself is unverified; `vcs.pb.go` was not regenerated.
