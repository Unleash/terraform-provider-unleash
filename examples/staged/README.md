# About this project
This is a sample repo that shows the evolution of an Unleash installation from simple to a more complex one.

If you just want to look at what Unleash managed with Terraform looks like, check out [stage 4](./stage_4)

## How to use this repo
### Prerequisites
- [Terraform](https://www.terraform.io/downloads.html) >= 0.13
- [Docker](https://docs.docker.com/get-docker/) >= 20.10.7
- [Docker compose V2](https://docs.docker.com/compose/cli-command/) >= 2.0.0
- [Bash](https://www.gnu.org/software/bash/) >= 5.1.8
- Access to [Unleash Enterprise](https://www.unleash-hosted.com/) docker image

### Running the steps
A makefile makes it easy to run the steps. The steps are numbered and can be run in order. The steps are also described in the sections below.

By default it will leave an instance of Unleash running at localhost:4242 so you can check the results. If you want to clean up the docker containers, run `make clean`.

Note that every time you run make it will perform a cleanup before running the steps.

* Run `make` to run all the steps
```shell
make
```

* Run each one of the steps individually (useful for debugging)
```shell
make STEPS="step_1"
# or
make STEPS="step_1 step_2"
# because the steps are stateless, you can run them in any order (one by one or just execute any)
make STEPS="step_4"
```

* Add debug information
```shell
make TF_LOG=debug
```


## Stages
### Stage 1
Create a first project and a custom root roles

### Stage 2
Incorporates project roles and a few users.
It also imports the default project that can then be used later on.

### Stage 3
Removes one user and creates some api tokens, modifies users and projects

### Stage 4
Reorganizes everything. It removes old projects and creates new ones. It also creates new users and api tokens.
It defines a module to simplify the creation of projects and uses locals to declare what it wants to create.
