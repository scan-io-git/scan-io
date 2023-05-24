# Goal

This document describe how to run analysis in kubernetes.  
This would allow to speed up scan of many projects by scaling cluster, adjusting number of nodes and scanning jobs.

# Prerequisites

1. Prepare k8s infrastructure: AWS EKS, minikube or any other k8s cluster.
2. Install and configure kubectl
3. Get familiar with [helm](https://helm.sh). Install helm CLI, which will reuse kubeconfig

# Why helm?
The first approach for remote runtime was to use kubernetes go client code. It was pretty native approach, although not flexibale enough. Many compnaies have unique infrastructure and require custom configuration. For example, use hashicorp vaulnt instead of kubernetes secrets, persistant storage configuration, custom admission controllers, etc. Helm use templates engine, which allows to describe all required kubernetes resources to run in a cluster.  
The following guide describe some basic helm chart usage. And you can always customize it.

# Run remote scan with helm runtine
```bash
# List repos
❯ scanio list --vcs github --vcs-url github.com --namespace juice-shop --output /tmp/juice-shop-projects.json

# Run scan with "helm" runtime
# By default scanio get helm chart from "./helm/scanio-job" folder
❯ scanio run2 --auth-type http -f /tmp/juice-shop-projects.json --scanner bandit --vcs github --runtime helm -j 5

# It's always good idea to review "./helm/scanio-job/values.yaml" to update some variables to fit your environment requirements.

# in case of using pvc, you can run `sleep infinity` pod and access fs with results:
❯ helm install scanio-main-pod helm/scanio-main-pod/
❯ kubectl exec -it scanio-main-pod -- bash

# By default scanio saves results to `/data/results/` folder with respect to `vcs-url`, `namespace` and `repo-name`. For example
❯ ls /data/results/github.com/juice-shop/juice-shop/bandit.raw 
/data/results/github.com/juice-shop/juice-shop/bandit.raw
```
