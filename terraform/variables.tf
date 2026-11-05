variable "cluster_name" {
  description = "Name of Cluster"
  type        = string
  default     = "istio-meshguard"
}

variable "node_iam_role_name" {
  description = "Name of Node IAM role"
  type        = string
  default     = "istio-meshguard-eks-node"
}

variable "project_name" {
  type    = string
  default = "istio-meshguard"
}