# How to get scanio cli

Currently, there are just few ways to get and start using scanio cli.  

## From a source code
The first option is to clone the repo and build scanio core and plugins from a source code:  
```
git clone https://github.com/scan-io-git/scan-io
cd scan-io
make build
```

## As a docker image
Another option is to use a docker image. You can fetch one of the images we build:
```
docker run --rm ghcr.io/scan-io-git/scan-io:latest --help
```  

Or you can build it from sources with `make docker` command.
