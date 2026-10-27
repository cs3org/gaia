# gaia | custom reva binary builder

<img src="assets/logo.png" width="200" alt="gopher with a wrench in his hand">

gaia simplifies the process of creating custom [reva](https://reva.link/) binaries with specific plugins. This tool automates the compilation and packaging of reva with selected plugins, making it easier for users to create tailored deployments.

## Features

- Easy selection of plugins to include in the custom reva binary.
- Automated compilation and packaging of the custom binary.
- Streamlined deployment process for reva with selected plugins.

## Installation

```
go install github.com/cs3org/gaia@latest
```

## Usage

```
gaia build [<reva_version>]
    [--binary revad|reva]
    [--with <module[/package][@version][=replacement]>...]
    [--brand <file.toml>]
    [--output <file>]
```

By default, gaia use the latest available version of reva.

### The program

`--binary` picks the program to build:

- `revad`, the reva daemon (the default). The output is `./revad`.
- `reva`, the reva CLI, with the CLI plugins (`cli.*`) of the build. The output is `./reva`.

### Plugins

`--with` adds a plugin: a module, or a package inside a module. The replacement is a module (`path@version`), or a
local directory:

```
gaia build --with github.com/cernbox/reva-plugins=../reva-plugins
gaia build --binary reva --with github.com/cernbox/reva-plugins/cli=../reva-plugins/cli
```

For a local directory, gaia reads the module path from the `go.mod` at or above it, and replaces that module. The
directory is the root of the module or the directory of the package.

`--with github.com/cs3org/reva/v3=../reva` builds a local reva.

### The brand of the reva CLI

`--brand <file.toml>` (only with `--binary reva`) builds the CLI with a brand: its program name, its defaults, its
environment variables and its texts. The schema is reva's `pkg/cli/brand`. gaia copies the file into the workspace as
`brand.toml`, and `main.go` embeds it.

```
gaia build --binary reva --brand cernbox.toml --with github.com/cernbox/reva-plugins/cli -o ./cernbox
```

Before the build, gaia runs `go run . __brandcheck` in the workspace, for every reva build. reva checks its brand
strictly, against the plugins of the same build, and checks the plugins. A bad brand or a refused plugin fails the
build.

### Version

gaia sets the version, the commit, the Go version and the build date of reva, in `github.com/cs3org/reva/v3/pkg/rversion`
(and in `cmd/revad`, for the older releases).

### Building the workspace with other tools

`--only-prepare` stops after the prepare stage, and keeps the workspace (`--workspace`). The workspace is a Go module
with a `main.go`, and the file `bflags` holds the version ldflags. Another tool can build it, for any platform:

```
gaia build --only-prepare --workspace ./ws --binary reva --brand cernbox.toml --with github.com/cernbox/reva-plugins/cli
cd ws && GOOS=darwin go build -trimpath -ldflags "-s -w $(cat bflags)" -o cernbox .
```

`--only-build --workspace ./ws` builds a prepared workspace. Pass the same `--binary`, or `--output`.

### Tests

The build tests compile reva from a local checkout, and are skipped without one:

```
GAIA_TEST_REVA=/path/to/reva go test ./pkg/builder
```
