# container vars
variable "registry_url" {
  description = "Docker Registry URL to use for this deployment.  Blank will default to Public Docker Hub Registry"
  default = "315451807861.dkr.ecr.us-east-2.amazonaws.com/"
}
variable "namespace" {
  description = "Docker image namespace to use for this deployment. Blank namespace for official docker images (library). For user repositories you need namespace/"
  default = ""
}
variable "container_name" {
  description = "Docker image container names in a csv list"
  default = "conscience-node"
}
variable "container_p2p_port" {
  description = "P2P port inside of container[i] (comma separated list)"
  default = "1337"
}
variable "container_rpc_port" {
  description = "RPC port inside of container[i] (comma separated list)"
  default = "1338"
}
variable "container_http_port" {
  description = "Application port inside of container[i] (comma separated list)"
  default = "8081"
}
variable "desired_count" {
  description = "Desired number of containers running for each service (comma separated list)"
  default = "1"
}
variable "version_tag" {
  description = "Docker image version tag to use for this deployment (comma separated list)"
  default = "latest"
}
variable "health_check" {
  description = "ALB Health-Check for Microservice, Defaults to / (comma separated list)"
  default = ""
}
variable "env_key" {
  description = "Additional environment variable key of value to set in containers.  Cannot be blank"
  default = "key"
}
variable "env_value" {
  description = "Additional environment variable value of key to set in containers.  Cannot be blank"
  default = "value"
}

variable "add_aws_policy" {
  description = "Attach additional Managed Policy to the ECS service?"
  default = false
}
variable "aws_policy" {
  description = "AWS Manged Policy to attach to ECS service, e.g. AmazonDynamoDBReadOnlyAccess"
  default = ""
}
