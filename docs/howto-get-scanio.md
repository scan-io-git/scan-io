# How to get Scanio CLI

To start using Scanio CLI, there are a few straightforward methods available.

## From Source Code
The first option involves cloning the repository and building the Scanio core and plugins from the source code. Follow these steps:  
```
git clone https://github.com/scan-io-git/scan-io
cd scan-io
make build
```

## Using a Docker Image
Alternatively, you can utilize a Docker image to quickly get Scanio CLI up and running. Pull one of the pre-built images:
```
docker run --rm ghcr.io/scan-io-git/scan-io:latest --help
```  

Or build the Docker image from the sources with the following command: `make docker`.

## With Go Install
It's also possible to use the built-in Go feature to install projects. Use this command:
```
go install github.com/scan-io-git/scan-io@latest
```
This command installs Scanio CLI and all its dependencies. The tool can be executed by calling `scan-io --help`.  
> Note that this method installs only the Scanio CLI core, not the plugins.
