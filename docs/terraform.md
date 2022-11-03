# Purpose of this document

Describe to easily configure AWS EKS infrastructure by reusing terraform files in `/terraform` folder. Every company has unique environment and kubernetes customizations, but this one is handy for enrolling dev environment.

# Prerequisites
1. Get familiar with [AWS EKS](https://aws.amazon.com/eks/) and [AWS IAM](https://aws.amazon.com/iam/).
2. Get `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`. (How to create service account with limited rights is out of scope of this document.). Configure AWS CLI.
3. Get familiar with [Terraform](https://developer.hashicorp.com/terraform/intro) and install Terraform CLI
4. Install kubectl, we will configure it later after k8s cluster creation.
5. Install docker to build and push image.

# Usage
Before deploying EKS it's a good idea to review terraform files.  
Configure your environment by modifying `variables.tf` file.  
If you need more control it's a good idea to create new variables and create a Merge Request.

## Commands to know
```bash

# Go to '/terraform' folder.
cd ./terraform

# init terraform before the usage (needs to be done once)
terraform init

# create or update infrastructure: EKS cluster + ECR
# TAKES ABOUT 15-20 MINUTES
terraform apply

# retrieve the access credentials for your cluster and configure kubectl
aws eks --region $(terraform output -raw region) update-kubeconfig --name $(terraform output -raw cluster_name)

# Now we have to configure docker credentials to authenticate against AWS ECR.
# You can find the command on a apge of previously created ECR cluster.

# get repository url
export DOCKER_IMAGE=$(terraform output -raw repository_url)
# This is an image name we have to use, to push and docker image
# `docker build -t $DOCKER_IMAGE . && docker push $DOCKER_IMAGE`

# After you finished working with the infrastructure, destroy it. May take about 15-20 minutes.
terraform destroy
# During execution of `apply` command, terraform create about 50 resources. Deleting them manually is painfull. Terraform store state with all deployed resources in a state file. By executing `terraform destroy` on the same machine will use this state file, and determine resources to be deleted automatically.
```
