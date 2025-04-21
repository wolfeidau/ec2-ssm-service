# Define variables
variable "app_name" {
  type        = string
  description = "The name of the application."
  default     = "demo"
}

variable "stage" {
  type        = string
  description = "The stage where the application is running in, e.g., dev, prod."
  default     = "dev"
}

variable "branch" {
  type        = string
  description = "The branch name"
  default     = "main"
}

variable "component" {
  type        = string
  description = "The component of the application, e.g., api, web, worker."
  default     = "ci"
}

variable "buildkite_token" {
  type        = string
  description = "The Buildkite token for the agent."
  sensitive   = true
}

variable "honeycomb_api_key" {
  type        = string
  description = "The Honeycomb API key for the application."
  sensitive   = true
}

# Create SSM parameters
resource "aws_ssm_parameter" "buildkite_agent_config" {
  name        = "/config/${var.stage}/${var.branch}/${var.app_name}/${var.component}/config/buildkite-agent-cfg"
  type        = "SecureString"
  value       = <<-EOT
    # The token from your Buildkite "Agents" page
    token="${var.buildkite_token}"

    # The name of the agent
    name="%hostname-%spawn"

    # Path to where the builds will run from
    build-path="/var/lib/buildkite-agent/builds"

    # Directory where the hook scripts are found
    hooks-path="/etc/buildkite-agent/hooks"

    # When plugins are installed they will be saved to this path
    plugins-path="/etc/buildkite-agent/plugins"
  EOT
  description = "Buildkite Agent Config"
}

resource "aws_ssm_parameter" "honeycomb_api_key_env" {
  name        = "/config/${var.stage}/${var.branch}/${var.app_name}/${var.component}/env/OTEL_EXPORTER_OTLP_HEADERS"
  type        = "SecureString"
  value       = "x-honeycomb-team=${var.honeycomb_api_key}"
  description = "Honeycomb API Key Environment Variable"
}

resource "aws_ssm_parameter" "honeycomb_service_name_env" {
  name        = "/config/${var.stage}/${var.branch}/${var.app_name}/${var.component}/env/OTEL_SERVICE_NAME"
  type        = "String"
  value       = "${var.app_name}-${var.component}"
  description = "Honeycomb Service Name Environment Variable"
}

resource "aws_ssm_parameter" "honeycomb_endpoint_env" {
  name        = "/config/${var.stage}/${var.branch}/${var.app_name}/${var.component}/env/OTEL_EXPORTER_OTLP_ENDPOINT"
  type        = "String"
  value       = "https://api.honeycomb.io"
  description = "Honeycomb Endpoint Environment Variable"
}