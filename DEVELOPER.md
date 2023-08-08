# Developer Requirements

* [Terraform (Core)](https://www.terraform.io/downloads.html) - version 1.x or above
* [Go](https://golang.org/doc/install) version 1.20.x (to build the provider plugin)

## Contributor Guides

A Collection of guides geared towards contributors can be found in the [`/contributing`](https://github.com/Azure/terraform-provider-alz/tree/main/contributing) directory of this repository.

### On Windows

If you're on Windows you'll also need:

* [Git Bash for Windows](https://git-scm.com/download/win)
* [Make for Windows](http://gnuwin32.sourceforge.net/packages/make.htm)

For *GNU32 Make*, make sure its bin path is added to PATH environment variable.*

For *Git Bash for Windows*, at the step of "Adjusting your PATH environment", please choose "Use Git and optional Unix tools from Windows Command Prompt".*

Or install via [Chocolatey](https://chocolatey.org/install) (`Git Bash for Windows` must be installed per steps above)

```powershell
choco install make golang terraform -y
refreshenv
```

You must run `Developing the Provider` commands in `bash` because `sh` scrips are invoked as part of these.

You may hit issues with `make build` telling you every file needs to be formatted as a result of line endings. To avoid this issue set your git config using `git config --global core.autocrlf false`. This will tell git to use the source `LF` rather than the Windows default of `CRLF`.

You may get errors when cloning the repository on Windows that end with `Filename too long`. To avoid this issue set your git config using `git config --system core.longpaths true`. This will tell git to allow file names longer than 260 characters which is the default on Windows.

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine. You'll also need to correctly setup a [GOPATH](http://golang.org/doc/code.html#GOPATH), as well as adding `$GOPATH/bin` to your `$PATH`.

First clone the repository.

Once inside the provider directory, you can run `make tools` to install the dependent tooling required to compile the provider.

At this point you can compile the provider by running `make build`, which will build the provider and put the provider binary in the `$GOPATH/bin` directory.

```sh
$ make build
...
$ $GOPATH/bin/terraform-provider-azurerm
...
```

You can also cross-compile if necessary:

```sh
GOOS=windows GOARCH=amd64 make build
```

In order to run the `Unit Tests` for the provider, you can run:

```sh
make test
```

The majority of tests in the provider are `Acceptance Tests` - which provisions real resources in Azure. It's possible to run the entire acceptance test suite by running `make testacc` - however it's likely you'll want to run a subset, which you can do using a prefix, by running:

```sh
make testacc
```

**Note:** Acceptance tests read data from Azure and need a valid authorization context. You can log in using `az cli` to do this.

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

### Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

---

## Developer: Using the locally compiled Azure Provider binary

After successfully compiling the Azure Provider, you must [instruct Terraform to use your locally compiled provider binary](https://www.terraform.io/docs/commands/cli-config.html#development-overrides-for-provider-developers) instead of the official binary from the Terraform Registry.

For example, add the following to `~/.terraformrc` for a provider binary located in `/home/developer/go/bin`:

```hcl
provider_installation {

  # Use /home/developer/go/bin as an overridden package directory
  # for the azure/alz provider. This disables the version and checksum
  # verifications for this provider and forces Terraform to look for the
  # azurerm provider plugin in the given directory.
  dev_overrides {
    "azure/alz" = "/home/developer/go/bin"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
```

---

## Developer: Scaffolding the Website Documentation

You can scaffold the documentation by running:

```sh
make docs
```
