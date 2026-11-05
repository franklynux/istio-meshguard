
output "vpc_id" {
  value       = aws_vpc.cluster_vpc.id
  description = "VPC ID"

}

output "public_subnets" {
  value = aws_subnet.pub-subnet[*].id

}

output "private_subnets" {
  value = aws_subnet.priv-subnet[*].id

}

