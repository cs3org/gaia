# gaia - custom reva binary builder

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
    [--with <module[@version][=replacement]>...]
    [--output <file>]
```

By default, gaia use the latest available version of reva.