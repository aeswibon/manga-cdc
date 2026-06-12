output "region" {
  value = var.region
}

output "tf_state_bucket" {
  value = aws_s3_bucket.tf_state.bucket
}

output "aws_role_arn" {
  value = aws_iam_role.github_deploy.arn
}

output "aws_account_id" {
  value = data.aws_caller_identity.current.account_id
}
