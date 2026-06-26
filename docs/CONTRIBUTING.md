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

## Releasing new versions

### Prerequisites for releasing

1. ensure you have [goreleaser](https://goreleaser.com/) installed:

  ```bash
  brew install goreleaser/tap/goreleaser
  ```

2. Create a new token for goreleaser [in your GitHub settings](https://github.com/settings/tokens); select the `repo` scope.

3. put the just-created token into the file `~/.config/goreleaser/github_token`


### Doing the release

Testing a release:

```
goreleaser release --snapshot --skip=publish --clean --verbose
```

Executing a release:

1. Commit all changes, create a new tag and push it.

```
TAG=v0.9.0; git tag $TAG; git push origin $TAG
```

2. run goreleaser:

```
goreleaser release --clean
```

