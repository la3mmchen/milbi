# 02 – GitHub workflows and release

## Behaviour

**`.github/workflows/ci.yaml`** runs on every push and pull request:

- `golangci-lint`
- `task test` on `ubuntu-latest` and `macos-latest`
- `task test:integration` on `ubuntu-latest`, with restic and rsync installed via apt

**`.github/workflows/release.yaml`** runs on tags matching `v*.*.*`:

- It runs the same lint and test jobs first. The release job `needs` them.
- GoReleaser builds `darwin` and `linux` binaries for `amd64` and `arm64`.
- It publishes `tar.gz` archives plus `checksums.txt` to a GitHub Release, with a changelog generated from commits.
- The workflow has `permissions: contents: write` on the release job only, and read-only elsewhere.
- Third-party actions are pinned to a full commit SHA, with the version noted in a comment.

## Acceptance criteria

- [X] A pull request with a failing test shows a red CI check.
- [ ] Pushing tag `v0.1.0` creates a GitHub Release with four archives and a checksums file.
- [ ] A downloaded binary prints `v0.1.0` from `milbi version`.
- [ ] A failing test on the tag prevents the release from being created.
