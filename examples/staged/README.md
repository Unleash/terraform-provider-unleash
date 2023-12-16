# About this project
This is a sample repo that shows the evolution of an Unleash installation, starting from a simpler configuration and evolving to a more complex one.

If you just want to look at what Unleash managed with Terraform looks like, check out [stage 4](./stage_4)

## How to use this repo
### Prerequisites
- [Terraform](https://www.terraform.io/downloads.html) >= 0.13
- [Docker](https://docs.docker.com/get-docker/) >= 20.10.7
- [Docker compose V2](https://docs.docker.com/compose/cli-command/) >= 2.0.0
- [Bash](https://www.gnu.org/software/bash/) >= 5.1.8
- Access to [Unleash Enterprise](https://www.getunleash.io/) docker image

### Running the stages
A makefile makes it easy to run the stages. The stages are numbered and can be run in order. The stages are also described in the sections below.

By default it will leave an instance of Unleash running at http://localhost:4242 so you can check the results. If you want to clean up the docker containers, run `make clean`.

Note that every time you run `make` it will perform a cleanup before running the stages.

* Run `make` to run all the stages
```shell
make
```

* Run each one of the stages individually (useful for debugging)
```shell
make STAGES="stage_1"
# or
make STAGES="stage_1 stage_2"
# because the stages are stateless, you can run them in any order (one by one or just execute any)
make STAGES="stage_4"
```

* Add debug information
```shell
make TF_LOG=debug
```


## Stages
### [Stage 1](./stage_1)
Creates a first project and some custom root roles.

### [Stage 2](./stage_2)
Incorporates project roles and a few users.
It also imports the default project that can then be used later on.

### [Stage 3](./stage_3)
Removes one user and creates some API tokens. Modifies users and projects.

### [Stage 4](./stage_4)
Reorganizes everything. It removes old projects and creates new ones. It also creates new users and API tokens.
It defines a module to simplify the creation of projects and uses locals to declare what it wants to create.
