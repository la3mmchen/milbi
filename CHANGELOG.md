# changelog

## 0.1

- first version that might run on other notebooks than mine.

## 1.0

- releases via PyPi

## 2.0

The release replaces Python with Golang.

### Major changes

Switched to a new config format from a single custom yaml file to a config that borrows its format from Kubernetes manifests.

## Unreleased

Rewrite from scratch; see [specs/](specs/README.md). The config format changes again (`version: 1` with repositories, backups and syncs). The previous implementation is available under the `2.0.x` tags.
