variable "issuer_url" {
  type = string
}

variable "scope" {
  type = string
}

variable "azure_tenant_id" {
  type = string
}

variable "subscription_id" {
  type = string
}

variable "oidc_aud" {
  type = string
}

variable "oidc_sub" {
  type = string
}

provider "azurerm" {
  subscription_id                 = var.subscription_id
  resource_provider_registrations = "none"
  features {}
}

provider "azuread" {
  tenant_id = var.azure_tenant_id
}

output "tenant_id" {
  value = var.azure_tenant_id
}

resource "azuread_application" "example" {
  display_name = "cloudfed"

  api {
    requested_access_token_version = 2
  }

  web {
    redirect_uris = ["https://placeholder.local/callback"] # TODO: is this needed?
  }
}

resource "azuread_application_federated_identity_credential" "example" {
  application_id = azuread_application.example.id
  display_name   = "cloudfed"
  audiences = [var.oidc_aud]
  issuer         = var.issuer_url
  subject        = var.oidc_sub
  description    = "cloudfed example"
}

resource "azuread_service_principal" "example" {
  client_id = azuread_application.example.client_id
}

resource "azurerm_role_assignment" "example_reader" {
  scope                = var.scope
  role_definition_name = "Reader"
  principal_id         = azuread_service_principal.example.object_id
}

output "scope" {
  value = azurerm_role_assignment.example_reader.scope
}

output "client_id" {
  value = azuread_application.example.client_id
}
