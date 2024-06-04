# docfusion

## Prerequisites

- [ASDF](https://asdf-vm.com/)
  - [golang plugin](https://github.com/asdf-community/asdf-golang)
- [Docker](https://docs.docker.com/engine/install/)
  - [Compose](https://docs.docker.com/compose/install/)

### Docker Registry Access

## Setup

```sh
# install task runner
go install github.com/go-task/task/v3/cmd/task@latest
# reshim ASDF after installing global packages
asdf reshim golang
```

```sh
# install global dependencies (atlas, jet, air)
task install_global
# copy env file
task setup
```

Make sure GOPATH/GOROOT are set and $GOPATH/bin is in your PATH, e.g.

```sh
export GOPATH=$(asdf where golang)/packages
export GOROOT=$(asdf where golang)/go
#export PATH="${PATH}:$(go env GOPATH)/bin"
```

## Development

First, start the docker stack

```sh
task docker
```

Then, you can start the dev server.

```sh
task dev
```

When opening the `schema.hcl` in vscode, use Ctrl+Shift+P -> "Change Language Mode" -> "Atlas PostgreSQL" (using the `ariga.atlas-hcl` extension).

The [playground](https://gh.atlasgo.cloud/explore) can also be helpful for visualizing the schema.

## Testing

The test task should trigger all test commands.

```sh
task test
```

### Linting

```sh
task test:lint
```
