# Purpose

This is a guide how to scan all open source projects of specific organization on github.  
These instructions should be a base for scanning you private repos. 

# Steps
1. **`list` command**  
    Collect all projects of specific organization. For our example we would take OWASP's [ juice-shop](https://github.com/juice-shop).  
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
    # extract http_links from `juice-shop-projects.json` file.
    ❯ cat /tmp/juice-shop-projects.json | jq .result[].http_link | sed -e 's#^"https://github.com/##g' | sed -e 's#.git"$##g' > /tmp/juice-shop-projects-links.txt

    ❯ head -n 3 /tmp/juice-shop-projects-links.txt
    juice-shop/juice-shop
    juice-shop/pwning-juice-shop
    juice-shop/juice-shop-ctf

    # fetch all projects multiple threads
    ❯ scanio fetch --vcs github --vcs-url github.com --auth-type http -f /tmp/juice-shop-projects-links.txt -j 10
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
    # prepare path on fs to every project
    ❯ cat /tmp/juice-shop-projects.json | jq .result[].http_link | sed -e 's#^"https://##g' | sed -e 's#.git"$##g' > /tmp/juice-shop-projects-local-paths.json

    ❯ head -n 3 /tmp/juice-shop-projects-local-paths.json
    github.com/juice-shop/juice-shop
    github.com/juice-shop/pwning-juice-shop
    github.com/juice-shop/juice-shop-ctf

    # run scan with semgrep in multiple threads
    ❯ scanio analyse --scanner semgrep -f /tmp/juice-shop-projects-local-paths.json -j 10
    ```
    By default scanio save all results in `$HOME/.scanio/results` folder. For every project with respest to `vcs-url`, `namespace` and `repo_name`.
    ```bash
    ❯ cat ~/.scanio/results/github.com/juice-shop/juice-shop/semgrep.raw | jq .runs[0].results[50] | jq ".ruleId, .locations[0].physicalLocation.artifactLocation.uri, .locations[0].physicalLocation.region.snippet.text"
    "javascript.express.security.injection.tainted-sql-string.tainted-sql-string"
    "/Users/eprotsenko/.scanio/projects/github.com/juice-shop/juice-shop/data/static/codefixes/dbSchemaChallenge_3.ts"
    "    models.sequelize.query(`SELECT * FROM Products WHERE ((name LIKE '%${criteria}%' OR description LIKE '%${criteria}%') AND deletedAt IS NULL) ORDER BY name`)"
    ```
