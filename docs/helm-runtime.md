# Goal

This document describe how to run analysis in kubernetes.  
This would allow to speed up scan of many projects by scaling cluster, adjusting number of nodes and scanning jobs.

# Prerequisites

1. Install and configure AWS CLI.
2. Install and configure [kubectl](https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html)
3. Get familiar with [helm](https://helm.sh). Install helm CLI, which will reuse kubeconfig
4. Deploy infrastructure: AWS EKS, AWS ECR, AWS S3.

# Why helm?
The first approach for remote runtime was to use kubernetes go client code. It was pretty native approach, although not flexibale enough. Many compnaies have unique infrastructure and require custom configuration. For example, use hashicorp vaulnt instead of kubernetes secrets, persistant storage configuration, custom admission controllers, etc. Helm use templates engine, which allows to describe all required kubernetes resources to run in a cluster.  
The following guide describe some basic helm chart usage. And you can always customize it.

# Configuration
1. Configure AWS CLI "default" profile to have rights to work with EKS and push docker images to ECR.
2. Configure kubectl. For example:
```bash
aws eks --region $(terraform output -raw region) update-kubeconfig --name $(terraform output -raw cluster_name)
```
3. Configure AWS CLI "s3" profile to have rights to downloads scan results from s3. S3 is used as communication mechanism. Remote job upload scan results to s3, and local scanio core downloads these results.
4. Create kubernetes secret `s3` with `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`, which will be used by remote job to upload result to s3.  
```bash
❯ kubectl create secret generic s3 \
  --from-file=aws_access_key_id=/tmp/aws_access_key_id \
  --from-file=aws_secret_access_key=/tmp/aws_secret_access_key
```
5. Build docker image and push. For example:
```bash
❯ export DOCKER_IMAGE=$(terraform output -raw repository_url)

❯ docker build -f dockerfiles/Dockerfile -t $DOCKER_IMAGE .
```
6. Configure ECR (you can get instructions from aws console) and push the image. Example:
```bash
❯ aws ecr get-login-password --region eu-west-2 | docker login --username AWS --password-stdin $(aws sts get-caller-identity --query "Account" --output text).dkr.ecr.eu-west-2.amazonaws.com

❯ docker push $DOCKER_IMAGE
```

# Run remote scan with helm runtine
```bash
# List repos
❯ scanio list --vcs github --vcs-url github.com --namespace juice-shop --output /tmp/juice-shop-projects.json

# Get only paths
❯ cat /tmp/juice-shop-projects.json | jq .result[].http_link | sed -e 's#^"https://github.com/##g' | sed -e 's#.git"$##g' > /tmp/juice-shop-projects-paths.json

# Run scan with "helm" runtime
# By default scanio get helm chart from "./helm/scanio-job" folder
❯ scanio run -f /tmp/juice-shop-projects-paths.json --runtime helm --scanner-plugin bandit -j 10 --vcs-plugin github --vcs-url github.com

# It's always good idea to review "./helm/scanio-job/values.yaml" to update some variables to fit your requirements.

# By default scanio saves results to `$HOME/results/` folder with respect to `vcs-url`, `namespace` and `repo-name`. For example
❯ cd $HOME/.scanio/results/github.com/juice-shop/juice-shop/
❯ ls
bandit.raw
```