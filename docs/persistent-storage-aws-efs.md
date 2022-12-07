# Goal

This docs describe how to deploy AWS EFS statically provisioned persistant storage in your AWS EKS cluster.  
This PV will be used by k8s to save scanio analysis results.

# Prerequisites

1. Ready to use AWS EKS Cluster.
2. Get familiar with helm runtime and its chart.

# Preparing EFS Persistent Storage

Before using PV we have to install EFS driver in our cluster.  
The easiest way to do it is to follow official docs: https://docs.aws.amazon.com/eks/latest/userguide/efs-csi.html.  
During following this guide you will create AWS Service Account with rights to manage EFS, install EFS driver, create EFS file system.  
Handy help commands:
```bash
# get AWS account ID
❯ aws sts get-caller-identity --query 'Account' --output text

# get EKS region (if you use terraform)
❯ terraform output -raw region

# get EKS cluster name (if you use terraform)
❯ terraform output -raw cluster_name

# replace values in trust-policy file.
❯ sed -i '' -e "s/111122223333/$(aws sts get-caller-identity --query 'Account' --output text)/g" /tmp/trust-policy.json
❯ sed -i '' -e "s/region-code/$(terraform output -raw region)/g" /tmp/trust-policy.json
```
You will you these values to correctly adjust configuration file to install EFS driver.

At the last section of the guide choose "Deploy a sample application - Static".  
You have to apply storage class, pv and claim before using EFS with pods.

# Adjust helm chart to use EFS PV

You can either update chart file directly or update values. Turn `.Value.persistence.enabled` to true, and update claimName and mountPath.

# Usage example
1. I have created pv claim with name `efs-claim`.
2. Go to `./helm/values.yaml` and update `persistence` section:
```yaml
pv:
  efs:
    enabled: true
    claimName: efs-claim
    mountPath: /data
```
3. List Repos
```bash
❯ scanio list --vcs github --vcs-url github.com --namespace juice-shop --output /tmp/juice-shop-projects.json
```
4. Get only paths
```bash
cat /tmp/juice-shop-projects.json | jq .result[].http_link | sed -e 's#^"https://github.com/##g' | sed -e 's#.git"$##g' > /tmp/juice-shop-projects-paths.json
```
5. Run scanio with helm runtime
```bash
scanio run --vcs-plugin github --vcs-url github.com -f /tmp/juice-shop-projects-paths.json --runtime helm --scanner-plugin bandit -j 10 --storage-type remote
```
6. After scan finished you can create helm chart, which by default run pod with command `sleep infinity` (when scanio run scan this value is overriden). After helm install release, you can get pod name and attach to it. Inspect PV.
```bash
# install chart with infinity pod
❯ helm install infinitypod helm/scanio-job/ --set image.repository=$DOCKER_IMAGE

# find pod name and attach
❯ kubectl get pods
NAME                     READY   STATUS    RESTARTS   AGE
scanio-job-uanng-c4gsh   1/1     Running   0          3s

❯ kubectl exec -it scanio-job-uanng-c4gsh -- bash

# inspect pv
root@scanio-job-uanng-c4gsh:/# ls /data/
projects/ results/

root@scanio-job-uanng-c4gsh:/# ls /data/projects/github.com/juice-shop/
juice-shop/        juice-shop-ctf/    juicy-chat-bot/    juicy-coupon-bot/  juicy-malware/     juicy-statistics/  pwning-juice-shop/

root@scanio-job-uanng-c4gsh:/# ls /data/results/github.com/juice-shop/**/bandit.raw | head -n 3
/data/results/github.com/juice-shop/juice-shop-ctf/bandit.raw
/data/results/github.com/juice-shop/juice-shop/bandit.raw
/data/results/github.com/juice-shop/juicy-chat-bot/bandit.raw

# Download all results to local host
root@scanio-job-uanng-c4gsh:/# tar czvf /tmp/results.tar.gz /data/results/
root@scanio-job-uanng-c4gsh:/# exit

❯ kubectl cp scanio-job-uanng-c4gsh:/tmp/results.tar.gz /tmp/results.tar.gz
❯ tar xzvf /tmp/results.tar.gz

# cleanup infinity pod
❯ helm uninstall infinitypod
```
