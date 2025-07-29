variable "tenant" {
  type    = string
  default = "example-tenant"
  # it's good practice to have tenant-specific issuer URLs to avoid the potential
  # for confused-deputy attacks. for example, these can arise in aws when a customer
  # creates an IAM role that trusts your OIDC IdP, but they forgot to include
  # conditions that limit role assumption to their customer-specific `sub`. by having
  # tenant-specific issuer URLs, this issue can't happen.
}

variable "azure_tenant_id" {
  type = string
}

variable "azure_subscription_id" {
  type = string
}

variable "gcp_organization_id" {
  type = string
}

variable "gcp_project_name" {
  type = string
}

module "idp" {
  source = "./idp"
}

locals {
  issuer_url = "${module.idp.issuer_base_url}/${var.tenant}"
}

output "key_id" {
  value = module.idp.key_id
}

output "issuer_url" {
  value = local.issuer_url
}

output "idp_key_id" {
  value = module.idp.key_id
}

module "azure" {
  source = "./azure"

  issuer_url      = local.issuer_url
  azure_tenant_id = var.azure_tenant_id
  subscription_id = var.azure_subscription_id
  scope           = "/subscriptions/${var.azure_subscription_id}"
  oidc_aud        = "api://AzureADTokenExchange"
  oidc_sub        = "example-sub"
}

output "azure_client_id" {
  value = module.azure.client_id
}

output "azure_tenant_id" {
  value = module.azure.tenant_id
}

module "gcp" {
  source = "./gcp"

  issuer_url      = local.issuer_url
  organization_id = var.gcp_organization_id
  project_name    = var.gcp_project_name
  oidc_sub        = "example-sub"
}

output "gcp_service_account" {
  value = module.gcp.service_account
}

output "gcp_audience" {
  value = module.gcp.audience
}

output "gcp_organization_id" {
  value = module.gcp.organization_id
}

module "aws" {
  source = "./aws"

  issuer_url = local.issuer_url
  oidc_aud   = "sts.amazonaws.com"
  oidc_sub   = "example-sub"
}

output "aws_role_arn" {
  value = module.aws.role_arn
}
