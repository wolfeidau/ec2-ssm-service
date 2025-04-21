output "buildkite_agent_config_arn" {
  value = aws_ssm_parameter.buildkite_agent_config.arn
}

output "honeycomb_api_key_env_arn" {
  value = aws_ssm_parameter.honeycomb_api_key_env.arn
}

output "honeycomb_service_name_env_arn" {
  value = aws_ssm_parameter.honeycomb_service_name_env.arn
}

output "honeycomb_endpoint_env_arn" {
  value = aws_ssm_parameter.honeycomb_endpoint_env.arn
}