terraform {
  required_providers {
    random = {
      source  = "hashicorp/random"
      version = "3.6.3"
    }
  }
}
variable "issuer_url" {
  type        = string
  description = "comes from the idp terraform output"
}

variable "organization_id" {
  type = string
}

variable "project_name" {
  type = string
}

variable "oidc_sub" {
  type = string
}

provider "google" {
  project = var.project_name
}

resource "random_pet" "pool" {}

resource "google_iam_workload_identity_pool" "oidc_pool" {
  provider = google

  workload_identity_pool_id = random_pet.pool.id
  display_name              = "Example OIDC Pool"
  description               = "OIDC federation pool for external identities"
  disabled                  = false
}

resource "google_iam_workload_identity_pool_provider" "oidc_provider" {
  provider = google

  workload_identity_pool_id          = google_iam_workload_identity_pool.oidc_pool.workload_identity_pool_id
  workload_identity_pool_provider_id = "example-provider"

  display_name = "cloudfed"
  description  = "cloudfed description"

  oidc {
    issuer_uri = var.issuer_url
  }

  attribute_mapping = {
    "google.subject" = "assertion.sub"
  }
}

resource "random_pet" "service_account" {}

resource "google_service_account" "viewer_sa" {
  account_id   = random_pet.service_account.id
  display_name = "Created by cloudfed"
}

output "service_account" {
  value = google_service_account.viewer_sa.email
}

output "audience" {
  value = "//iam.googleapis.com/${google_iam_workload_identity_pool_provider.oidc_provider.name}"
}

output "organization_id" {
  value = var.organization_id
}

# Grant organization-level viewer role to the SA
resource "google_organization_iam_member" "viewer_binding" {
  org_id = var.organization_id
  role   = "roles/viewer"
  member = "serviceAccount:${google_service_account.viewer_sa.email}"
}

resource "google_service_account_iam_member" "impersonation_binding" {
  service_account_id = google_service_account.viewer_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principal://iam.googleapis.com/${google_iam_workload_identity_pool.oidc_pool.name}/subject/${var.oidc_sub}"
}
