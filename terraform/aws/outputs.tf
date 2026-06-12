output "vm_public_ip" {
  description = "Public IP address of the deployed EC2 VM instance"
  value       = length(aws_instance.app_ec2) > 0 ? aws_instance.app_ec2[0].public_ip : null
}

output "eks_endpoint" {
  description = "EKS Kubernetes cluster API endpoint"
  value       = length(aws_eks_cluster.eks) > 0 ? aws_eks_cluster.eks[0].endpoint : null
}
