# Unleash Terraform Provider

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.19

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Running tests

Run tests

```shell
go test -count=1 -v ./...
```

Run acceptance tests which cover the provider and resources code

```shell
TF_ACC=1 go test -count=1 -v ./...
```

### Before pushing

- `golangci-lint run` to lint the code
- `go generate ./...` to update docs

## Using the provider

Fill this in for each provider

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

In order to run the full suite of Acceptance tests, run `make testacc`.

_Note:_ Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```

### Using a local registry

When you are developing a Terraform provider it is often very helpful to tell terraform to use your local copy of the provider, instead of trying to download a proper provider from the Terraform Registry.

To do this we need to create a file in our home directory called .terraformrc that contains a provider_installation section that looks something like this:

```
provider_installation {
  dev_overrides {
    "Unleash/unleash" = "/usr/local/go/bin/"
  }
  direct {}
}
```

> To find your bin path, run `go env` and look at the values for GOPATH and GOROOT. Usually binaries will be installed under `${GOPATH}/bin`.
