variable "region" {
  description = "AWS region"
  type        = string
  default     = "eu-west-2"  # eu-west-2  # us-west-2
}

# https://docs.aws.amazon.com/eks/latest/APIReference/API_Nodegroup.html#AmazonEKS-Type-Nodegroup-amiType
variable "ami_type" {
  description = "ami type for nodes"
  type        = string
  default     = "AL2_x86_64"
  # default     = "AL2_ARM_64"
}

variable "instance_types" {
  description = "node instance type"
  type        = string
  default     = "t3.small"  # m6g.medium for arm, t3.small for x86
  # default     = "m6g.medium"  # m6g.medium for arm, t3.small for x86
}
