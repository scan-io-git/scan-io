# Purpose

This is a guide how to scan all open source projects of specific organization on github.  
These instructions should be a base for scanning you private repos. 

# Steps
1. **`list` command**  
    Collect all projects of specific organization. For our example we would take OWASP's [juice-shop](https://github.com/juice-shop).  
    `scanio list --vcs github --vcs-url github.com --namespace juice-shop --output /tmp/juice-shop-projects.json`  
    For github "namespace" is an org name. For bitbucket it's a project name. For gitlab it may have nested structure, including subgroups.  
    The output of `list` command is a file with information about repositories inside a namespace. You can review the file. The most important part for next steps is a list of structures. These data will be later used by scanio for cloning projects and for other need. 
    ```bash
    ❯ cat /tmp/juice-shop-projects.json | jq .result[0]
    {
      "namespace": "juice-shop",
      "repo_name": "juice-shop",
      "http_link": "https://github.com/juice-shop/juice-shop.git",
      "ssh_link": "git@github.com:juice-shop/juice-shop.git"
    }
    ```

2. **`fetch` command**  
    To clone projects we will reuse file `juice-shop-projects.json`.
    ```bash
    # fetch all projects multiple threads
    ❯ scanio fetch --vcs github --vcs-url github.com --auth-type http -i /tmp/juice-shop-projects.json -j 10
    ```
    By default scanio clone in `$HOME/.scanio/projects` folder as a root. Path to every cloned object is comprised of `<vcs-url>/<namespace>/<repo_name>`:
    ```bash
    ❯ ls ~/.scanio/projects/github.com/juice-shop/ | head
    juice-shop/
    juice-shop-ctf/
    juicy-chat-bot/
    juicy-coupon-bot/
    juicy-malware/
    juicy-statistics/
    pwning-juice-shop/

    ❯ ls ~/.scanio/projects/github.com/juice-shop/juice-shop/ | head -n 5
    Dockerfile
    Dockerfile.arm
    Gruntfile.js
    LICENSE
    app.json
    ```

3. **`analyze` command**
    After scanio fetched all projects, we can run analysis tools. For example, semgrep.
    ```bash
    # run scan with semgrep in multiple threads
    ❯ scanio analyse --scanner semgrep -i /tmp/juice-shop-projects.json -j 10
    ```
    By default scanio save all results in `$HOME/.scanio/results` folder. For every project with respest to `vcs-url`, `namespace` and `repo_name`.
    ```bash
    ❯ tree ~/.scanio/results/github.com/juice-shop/
    /Users/eprotsenko/.scanio/results/github.com/juice-shop/
    ├── juice-shop
    │   └── semgrep.raw
    ├── juice-shop-ctf
    │   └── semgrep.raw
    ├── juicy-chat-bot
    │   └── semgrep.raw
    ├── juicy-coupon-bot
    │   └── semgrep.raw
    ├── juicy-malware
    │   └── semgrep.raw
    ├── juicy-statistics
    │   └── semgrep.raw
    └── pwning-juice-shop
        └── semgrep.raw
    ```
