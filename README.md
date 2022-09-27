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
For the beginning I decided to put all staff (core, plugins, shared code) in same repo. Later, I think, we will move plugins to separate folder.
### Easy start
We these 2 commands you will build plugin and run core with custom command, that communicate with plugin over rpc.
1. build plugin: `cd ~/scan-io/plugins/gitlab/ && make build`.
2. run core: `cd ~/scan-io && go run main.go fetch`.
You should see:
```sh
fetch called
...
2022-09-25T11:05:34.152Z [DEBUG] plugin: starting plugin: path=/home/japroc/.scanio/plugins/gitlab args=[/home/japroc/.scanio/plugins/gitlab]
...
Hello!
...
2022-09-25T11:05:34.160Z [DEBUG] plugin: plugin exited
```
