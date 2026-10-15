output "cluster_name" {
  value       = module.eks.cluster_name
  description = "Name of Hub Cluster"
}

output "cluster_endpoint" {
  value       = module.eks.cluster_endpoint
  description = "Cluster API endpoint"
}

output "cluster_certificate_authority_data" {
  value       = module.eks.cluster_certificate_authority_data
  description = "Cluster CA data for kubeconfig"
  sensitive   = true
}

output "ecr_repository_urls" {
  value = { for svc, m in module.ecr : svc => m.repository_url }
}