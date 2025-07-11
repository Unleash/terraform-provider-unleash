# Unleash Terraform Provider

[Join us on Slack](https://slack.unleash.run) at **#terraform-provider**

- Documentation on [terraform registry](https://registry.terraform.io/providers/unleash/unleash/latest/docs)

## Overview

This terraform provider is not intended to support everything in Unleash. The main focus is to support Unleash's initial setup and configuration.

Because [feature flags should be short-lived](https://docs.getunleash.io/topics/feature-flags/short-lived-feature-flags), we do not support managing feature flags through Terraform. Feature flags should be managed directly in Unleash.

If you're interested in using Terraform to manage feature flags, you can use [philips-labs/unleash provider](https://registry.terraform.io/providers/philips-labs/unleash/latest/docs) that supports managing feature flags.

Note that some resources are only available for the enterprise version of Unleash.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.19
- Unleash server v5.6.0

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Testing

https://developer.hashicorp.com/terraform/plugin/testing

**Note**: some tests rely on an enterprise version of Unleash. To run those tests locally you need to set the environment variable `UNLEASH_ENTERPRISE=true`. To run docker with an enterprise image: `UNLEASH_DOCKER_IMAGE=unleashorg/unleash-enterprise:latest docker compose up` (you will also need a valid license key that you can provide at startup with `UNLEASH_DEV_LICENSE=<your license key>`).

Run tests (most likely we will not have a lot of unit tests but instead we'll have acceptance tests)

```shell
go test -count=1 -v ./...
```

Run **acceptance tests** which cover the provider and resources code

```shell
TF_LOG=debug TF_ACC=1 go test ./... -v -count=1
```

To run enterprise tests (you have to make sure you're running an enterprise server)
```shell
UNLEASH_ENTERPRISE=true TF_LOG=debug TF_ACC=1 go test ./... -v -count=1
```

or the following make target (although it will cache results if nothing changes)

```shell
make testacc
```

### Before pushing

- `golangci-lint run --fix` to lint the code
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
