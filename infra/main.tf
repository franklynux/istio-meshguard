provider "aws" {
  region  = "us-east-1"
  profile = "franklynux"
}

locals {
  services = toset(["orders", "inventory", "payments", "notifications"])
}

# --- S3 Remote Backend ---
terraform {
  backend "s3" {
    bucket  = "fleetform-tfstate-franklynux"
    key     = "istio/terraform.tfstate"
    region  = "us-east-1"
    encrypt = true
    profile = "franklynux"
  }
}

# --- NODE IAM ROLE ---
resource "aws_iam_role" "node" {
  name = "istio-meshguard-eks-node"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "node_policies" {
  for_each = toset([
    "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
    "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
    "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly",
  ])

  role       = aws_iam_role.node.name
  policy_arn = each.value
}


module "vpc" {
  source     = "./modules/vpc"
  cidr_block = "10.0.0.0/16"
  vpc_name   = "istio-meshguard-vpc"
}

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 21.0"

  name                                     = var.cluster_name
  kubernetes_version                       = "1.32"
  enable_cluster_creator_admin_permissions = true
  endpoint_public_access                   = true

  vpc_id     = module.vpc.vpc_id
  subnet_ids = module.vpc.private_subnets
}

resource "aws_eks_addon" "pre_node" {
  for_each = toset(["vpc-cni", "kube-proxy", "eks-pod-identity-agent"])

  cluster_name = module.eks.cluster_name
  addon_name   = each.value

  depends_on = [module.eks]
}

resource "aws_eks_node_group" "this" {
  cluster_name    = module.eks.cluster_name
  node_group_name = "hub"
  node_role_arn   = aws_iam_role.node.arn
  subnet_ids      = module.vpc.private_subnets

  instance_types = ["t3.medium"]

  scaling_config {
    min_size     = 1
    max_size     = 3
    desired_size = 3
  }

  depends_on = [aws_eks_addon.pre_node, aws_iam_role_policy_attachment.node_policies]
}

resource "aws_eks_addon" "coredns" {
  cluster_name = module.eks.cluster_name
  addon_name   = "coredns"

  depends_on = [aws_eks_node_group.this]
}

module "ecr" {
  source               = "./modules/ecr"
  for_each             = local.services
  image_tag_mutability = "MUTABLE"

  name = "${var.project_name}/${each.key}"
}