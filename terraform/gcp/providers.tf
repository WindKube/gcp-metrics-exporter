terraform {
  required_version = ">= 1.5"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = ">= 5.0"
    }
  }
}

# The provider uses Application Default Credentials. For local use, run
# `gcloud auth application-default login`; production workloads should use
# their attached identity (for example, a GKE Workload Identity service account).
provider "google" {
  project = var.service_account_project_id
}
