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
It's also possible to use the built-in Go package installation feature. Use this command:
```
go install github.com/scan-io-git/scan-io@latest
```
This command installs Scanio CLI and all its dependencies. The tool can be executed by calling `scan-io --help`.  
Instead of installing the latest version, you can specify a specific version, such as:
```
go install github.com/scan-io-git/scan-io@v0.1.0
```
> Note: This method installs only the core of Scanio CLI, excluding the plugins.
