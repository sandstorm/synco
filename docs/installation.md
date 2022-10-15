# Quick Start - Install cli-template

> [!TIP]
> cli-template is installable via [instl.sh](https://instl.sh).\
> You just have to run the following command and you're ready to go!

<!-- tabs:start -->

#### ** Windows **

### Windows Command

```powershell
iwr instl.sh/sandstorm/synco/windows | iex
```

#### ** Linux **

### Linux Command

```bash
curl -sSL instl.sh/sandstorm/synco/linux | bash
```

#### ** macOS **

### macOS Command

```bash
curl -sSL instl.sh/sandstorm/synco/macos | bash
```

#### ** Compile from source **

### Compile from source with Golang

?> **NOTICE**
To compile cli-template from source, you have to have [Go](https://golang.org/) installed.

Compiling cli-template from source has the benefit that the build command is the same on every platform.\
It is not recommended to install Go only for the installation of cli-template.

```command
go install github.com/sandstorm/synco@latest
```

<!-- tabs:end -->
