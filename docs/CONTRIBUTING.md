# Contributing

We'd love to have pull requests or bug reports :-) In case we do not react timely,
do not hesitate to get in touch with us e.g. via [kontakt@sandstorm.de](mailto:kontakt@sandstorm.de),
as it might happen that a pull request slips through.

## Developing

Simply have a modern Go version installed; check out the project somewhere (NOT in $GOPATH, as we use Go Modules),
and then run `make`.

### ... with a public network

for testing with locally built synco-lite versions, you can upload your locally built synco:

```bash
# build synco for Linux (match the TARGET's CPU architecture, see note below)
GOOS=linux GOARCH=amd64 make synco-lite   # or GOARCH=arm64

# upload synco to the remote system
scp build/synco-lite .....
# or:
kubectl cp build/synco-lite POD-NAME-HERE:/tmp/synco-lite
```

> [!important]
> **Build for the target's CPU architecture.** Set `GOARCH` to match where the
> binary will run — `arm64` for an `aarch64` host/container (e.g. a Docker
> Desktop container on Apple Silicon), `amd64` for `x86_64`. Check with
> `uname -m` on the target (`docker compose exec <service> uname -m`).
>
> A mismatched binary runs under emulation, and emulated ChaCha20-Poly1305 SIMD
> (from `golang.org/x/crypto`) produces **wrong** output for multi-block
> payloads. The failure is misleading: `synco receive` reports a valid
> decryption key (the age header decrypts), then fails on the payload with
> `failed to decrypt and authenticate payload chunk`. It looks like corruption
> or a wrong password, but the real cause is the arch mismatch.

## Running the tests

```bash
# unit tests only (no external dependencies)
make test-unit

# full suite, including the end-to-end tests
make test
```

The end-to-end tests in `test_e2e/` spin up a MariaDB container via
[gnomock](https://github.com/orlangure/gnomock), so `make test` requires a
**running Docker daemon**. They reuse a container named `synco-test-flow`
between runs to stay fast; remove it with `docker rm -f synco-test-flow` if you
need a clean database.

If you don't have Docker available, run `make test-unit` instead — it skips the
`test_e2e` package and has no external dependencies.

## Releasing new versions

### Doing the release (via CI — the normal way)

Releases happen **automatically in GitHub Actions**. The
[`release` workflow](../.github/workflows/release.yml) is triggered by pushing a
tag and runs `goreleaser release --clean` for you, building all target platforms
and publishing the GitHub Release with the artifacts.

So a release is just a tag push:

```
TAG=v0.9.0; git tag $TAG; git push origin $TAG
```

> [!note]
> Do **not** also run `goreleaser release` locally for the same tag — CI already
> does it. Running it locally as well would collide with the CI run (both try to
> create the same GitHub Release and upload the same assets).

On every push and pull request, the
[`release-test` workflow](../.github/workflows/release-test.yml) additionally
runs the test suite and a snapshot build (`--snapshot --skip=publish`) so release
problems surface before you tag.

### Testing a release locally

To dry-run the release build locally (no publishing), you don't need any tokens:

```
goreleaser release --snapshot --skip=publish --clean --verbose
```

### Doing the release locally (fallback / reference)

Normally you should let CI publish (see above). If you ever need to publish a
release from your machine — e.g. CI is unavailable — you can run goreleaser
yourself:

1. ensure you have [goreleaser](https://goreleaser.com/) installed:

  ```bash
  brew install goreleaser/tap/goreleaser
  ```

2. Create a new token for goreleaser [in your GitHub settings](https://github.com/settings/tokens); select the `repo` scope.

3. put the just-created token into the file `~/.config/goreleaser/github_token`

4. Commit all changes, create a new tag and push it:

```
TAG=v0.9.0; git tag $TAG; git push origin $TAG
```

5. run goreleaser:

```
goreleaser release --clean
```

> [!important]
> Because pushing the tag already triggers the CI release, only publish locally
> if the CI run did not (or cannot) run for that tag — otherwise the two will
> conflict.

