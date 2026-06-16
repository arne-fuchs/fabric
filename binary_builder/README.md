# Binary Builder

An external builder and launcher that runs a **prebuilt chaincode executable**
shipped inside the chaincode package. No compilation happens on the peer and no
Docker is involved.

It is intended as a simple, out-of-the-box way to deploy chaincode while trying
out Fabric, and as part of the path to retiring the legacy docker-in-docker
build mechanism. See [hyperledger/fabric#3649](https://github.com/hyperledger/fabric/issues/3649).

> **Not recommended for production.** The peer runs whatever binary is in the
> package, as-is, on the peer's own platform. Use it for development,
> experimentation, and getting started.

## What goes in the chaincode package

A chaincode package is a `.tar.gz` containing `metadata.json` and a nested
`code.tar.gz`. For the binary builder, `code.tar.gz` must contain the prebuilt
executable:

```
<package>.tgz
├── metadata.json
└── code.tar.gz
    └── chaincode          # the prebuilt executable (name configurable, see below)
```

### metadata.json

```jsonc
{
  "type": "binary",            // selects this builder
  "label": "fabcar",
  "chaincodeData": {           // builder-specific config, nested to avoid key clashes
    "binary": "chaincode",     // optional: executable name in code.tar.gz (default "chaincode")
    "platform": "linux/amd64"  // optional: "<goos>/<goarch>" the binary targets
  }
}
```

- `type` **must** be `binary` for this builder to claim the package.
- `chaincodeData.binary` is optional; it defaults to `chaincode`.
- `chaincodeData.platform` is optional. When set, the build phase **fails with a
  clear error** if it does not match the peer's `GOOS/GOARCH`, instead of
  failing later with a cryptic `exec format error`. All peers in an organization
  must run a matching platform.

## What the binary must be

The executable is launched as a normal Fabric chaincode process. Before exec,
the builder provides connection details two ways, so both purpose-built and
standard chaincode work:

- **Standard shim environment** (works with unmodified Fabric chaincode):
  - argument `-peer.address=<addr>`
  - `CORE_CHAINCODE_ID_NAME`, `CORE_PEER_LOCALMSPID`
  - when TLS is enabled: `CORE_PEER_TLS_ENABLED=true` plus
    `CORE_TLS_CLIENT_CERT_FILE`, `CORE_TLS_CLIENT_KEY_FILE`,
    `CORE_PEER_TLS_ROOTCERT_FILE` pointing at PEM files written next to the
    launch directory (otherwise `CORE_PEER_TLS_ENABLED=false`).
- **`METADATA` environment variable**: the raw `chaincode.json` the peer
  supplies (`chaincode_id`, `peer_address`, `client_cert`, `client_key`,
  `root_cert`, `mspid`), for chaincode that prefers to parse it directly.

## Builder phases

The builder implements the standard external builder scripts as Go binaries:

| Command  | Peer invocation                          | Behaviour |
|----------|------------------------------------------|-----------|
| `detect` | `detect SOURCE_DIR METADATA_DIR`         | Exit 0 only when `type == "binary"`. |
| `build`  | `build SOURCE_DIR METADATA_DIR OUTPUT_DIR` | Validate type and platform, copy the executable to `OUTPUT_DIR/chaincode` (mode `0755`). |
| `run`    | `run OUTPUT_DIR LAUNCH_DIR`              | Read `LAUNCH_DIR/chaincode.json`, set up the environment above, and exec the binary. |

There is no `release` phase (none is needed; the peer skips it).

> Note: `run` does **not** receive `metadata.json` — only the build output and
> the peer connection info. Anything `run` needs is produced by `build`.

## Building

The builder ships compiled, like the CCaaS builder. From the Fabric repo root:

```sh
make binarybuilder            # build for the host platform
make binarybuilder/linux-amd64  # build for a specific platform
```

Binaries are written to `release/<platform>/builders/binary/bin/`. The
`docker`, `release`, and `dist` targets include it automatically.

To build manually:

```sh
cd binary_builder
for c in detect build run; do go build -o /path/to/builder/bin/ ./cmd/$c/; done
```

## Configuring a peer

Point the peer at the builder directory (the parent of `bin/`) in `core.yaml`:

```yaml
chaincode:
  externalBuilders:
    - name: binary
      path: /opt/hyperledger/binary_builder
```

When a `type: binary` package is installed, the peer's external builder
framework runs this builder's `detect`, `build`, and `run` in turn.

## Testing

```sh
cd binary_builder && go test ./...
```
