# Learn Terraform - Provision an EKS Cluster

This repo is a companion repo to the [Provision an EKS Cluster learn guide](https://learn.hashicorp.com/terraform/kubernetes/provision-eks-cluster), containing
Terraform configuration files to provision an EKS cluster on AWS.

# Commands to know:
```bash
# init terraform before the usage
terraform init

# create or update infrastructure (eks cluster)
# TAKES ABOUT 15 MINUTES
terraform apply

# retrieve the access credentials for your cluster and configure kubectl
aws eks --region $(terraform output -raw region) update-kubeconfig --name $(terraform output -raw cluster_name)

# get repository url
export DOCKER_IMAGE=$(terraform output -raw repository_url)
# dont forget to configure docker credentials for ecr
# and push image

# destroy infrastructure (eks)
terraform destroy
```
