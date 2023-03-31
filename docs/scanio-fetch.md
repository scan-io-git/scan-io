# Scanio Fetch Command
The main command's function is fetching code and consistency supporting (in a case when somebody deleted local files or files were corrupted - it works only if a .git folder wasn't deleted or corrupted). <br><br>

At the moment plugins fetch only the master/main branch.

## Args of the Command
- "vcs" is the plugin name of the VCS used. Eg. bitbucket, gitlab, github, etc.
- "input-file" or "f" is a file in scanio format with list of repositories to fetch. The list command could prepare this file.
- "auth-type" is a type for an authentication - "http", "ssh-agent" or "ssh-key".
- "ssh-key" is a path to an SSH key.
- "threads" or "j" is a number of concurrent goroutines. The default is 1. 
- "branch" or "b" is a specific branch for fetching. You can use it manual URL mode. A default value is main or master.
- "rm-ext" is a list of extensions that will be removed automatically after checkout. The default is  `csv,png,ipynb,txt,md,mp4,zip,gif,gz,jpg,jpeg,cache,tar,svg,bin,lock,exe`.<br><br>

Instead of using **input-file** flag you could use a specific **URL** that points to a particular namespace from your VCS (check the [link](#fetching-only-one-repo-manually-by-using-url)).

## Authentification Types Scenarios 
Plugins support three types of authentification:
- Using SSH keys as is.
- Using an SSH agent.
- Using an HHTP authentification.<br><br>

|Authentification type|Bitbucket|Gitlab|Github|
|----|-----|---|---|
|SSH keys|Supported|Supported for passwordless keys|Supported for passwordless keys|
|SSH agent|Supported|Supported|Supported|
|HTTP|Supported|Supported only anonymous access|Supported only anonymous access|


### SSH Keys
This method is using an SSH key from a disk.

#### Bitbucket 
For Bitbucket API v1 you need to use a few environment variables:
* SCANIO_BITBUCKET_SSH_KEY_PASSOWRD - your password for ssh. The default is an empty value!

#### Github
At the moment the plugin is working without any variables from an environment.

#### Gitlab
At the moment the plugin is working without any variables from an environment.

### SSH agent
This method is using an SSH key from a local ssh-agent.

#### Bitbucket
You should add your key to a local ssh-agent. <br>
```ssh-add /path/yourkey.private```

#### Github
You should add your key to a local ssh-agent. <br>
```ssh-add /path/yourkey.private```

#### Gitlab
You should add your key to a local ssh-agent. <br>
```ssh-add /path/yourkey.private```

### HTTP
This method is using the same token as for a list command and your username. 

#### Bitbucket
For Bitbucket API v1 you need to use a few environment variables:
* SCANIO_BITBUCKET_USERNAME - your username in BitBucket. **Mandatory**!
* SCANIO_BITBUCKET_TOKEN - your Bitbucket token. **Mandatory**!
  * It may be a plain text password or a personal access token from \<your_bb_domain\>/plugins/servlet/access-tokens/manage.

#### Github
At the moment the plugin is working without any variables from an environment.

#### Gitlab
At the moment the plugin is working without any variables from an environment.

## Using Scenarios 
When developing, we aimed at the fact that the program will be used primarily for automation purposes but you still able to use it manually from CLI.<br>

The command saves results into a home directory - ```~/.scanio/projects/+<VCSURL>+<Namespace>+<repo_name>```.<br>
You can redifine a home directory by using **SCANIO_HOME** environment variable.

### Fetching from Input File
The command uses an output format of a List command for fetching required repositories.

#### Bitbucket
* Fetching from an input file using an ssh-key authentification.<br>
```scanio fetch --vcs bitbucket --vcs-url example.com --input-file /Users/root/.scanio/output.file --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1```
* Fetching from an input file using an ssh-agent authentification.<br>
```scanio fetch --vcs bitbucket --vcs-url example.com --input-file /Users/root/.scanio/output.file --auth-type ssh-agent -j 1```
* Fetching from an input file with an HTTP authentification.<br>
```scanio fetch --vcs bitbucket --vcs-url example.com --input-file /Users/root/.scanio/output.file --auth-typ http -j 1```

#### Github
* Fetching from an input file using an ssh-key authentification.<br>
```scanio fetch --vcs github --vcs-url example.com --input-file /Users/root/.scanio/output.file --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1```
* Fetching from an input file using an ssh-agent authentification.<br>
```scanio fetch --vcs github --vcs-url example.com --input-file /Users/root/.scanio/output.file --auth-type ssh-agent -j 1```
* Fetching from an input file with an HTTP authentification.<br>
```scanio fetch --vcs github --vcs-url example.com --input-file /Users/root/.scanio/output.file --auth-typ http -j 1```

#### Gitlab
* Fetching from an input file using an ssh-key authentification.<br>
```scanio fetch --vcs gitlab --vcs-url example.com --input-file /Users/root/.scanio/output.file --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1```
* Fetching from input file with using an ssh-agent authentification.<br>
```scanio fetch --vcs gitlab --vcs-url example.com --input-file /Users/root/.scanio/output.file --auth-type ssh-agent -j 1```
* Fetching from input file with an HTTP authentification.<br>
```scanio fetch --vcs gitlab --vcs-url example.com --input-file /Users/root/.scanio/output.file --auth-typ http -j 1```

### Fetching only one repo manually by using URL
The command uses a link that is pointing to a particular repository for fetching.

#### Bitbucket
* Fetching using an ssh-key authentification and url that points to a specific repository.<br>
```scanio fetch --vcs bitbucket --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1 https://example.com/projects/scanio_project/repos/scanio/browse```

In this manual mode you can set a custom branch.<br>
```scanio fetch --vcs bitbucket --auth-type ssh-key --ssh-key /Users/root/.ssh/id_ed25519 -j 1 -b develop https://example.com/projects/scanio_project/repos/scanio/browse```

You can find additional information about URL formats [here](../plugins/bitbucket/README.md#supported-url-formats)

#### Github
*Not Supported*

#### Gitlab
*Not Supported*

## Possible Errors
### Bitbucket
#### ```ssh: handshake failed: knownhosts: key mismatch```
If you find the error check your .ssh/config. If you do use not a default 22 port for fetching and .ssh/config rules for this host, you have to determine a port too:
```
Host git.example.com
   Hostname git.example.com
   Port 7989 
   IdentityFile ~/.ssh/id_ed25519
``` 
Or just not use .ssh/config and the port will be identified automatically. 

#### ```ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain```
The algorithm is the same - determine a port in .ssh/config for your host or don't use .ssh/config rules.

#### ```Error on Clone occurred: err="reference not found"``` 
It means that a branch in a remote repo doesn't exits. 
Try to fix the name of the branch or project.

#### ```Error on Clone occurred: err="remote repository is empty"``` 
It means that a default branch in a remote repo (master/main) is empty.
Try to fix the name of the branch or project.

#### Github
*TODO*

#### Gitlab
*TODO*