# scan-io

## Bootstrap
### cobra
For cli management I used [cobra project](https://github.com/spf13/cobra).  
Fast intro to [cobra-cli and cli tool management](https://github.com/spf13/cobra-cli/blob/main/README.md).  
What I did:
```sh
mkdir ~/scan-io
cd ~/scan-io

go mod init github.com/scan-io-git/scan-io
cobra-cli init
cobra-cli add fetch
```
It created a skeleton of cli with "fetch" command, "--help" argument, etc...

### go-plugin
For plugin system I used [go-plugin library](https://github.com/hashicorp/go-plugin).  
This skeleton is heavily based on [basic example](https://github.com/hashicorp/go-plugin/tree/master/examples/basic). Examine it to understand what happens.

## Development
### Easy start
```sh
make clean
make buildplugins
make runprojects
make runorg
```