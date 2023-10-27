terraform {
  required_providers {
    unleash = {
      source  = "Unleash/unleash"
    }
  }
}

variable "id" {
  description = "The id of the project."
}

variable "name" {
  description = "The name of the project."
}

variable "owner" {
  description = "The owner of the project."
}

variable "members" {
  description = "List of ids of the members of the project."
  default = []
}

variable "tokens" {
  description = "List of tokens for the project."
  default = []
}

resource "unleash_project" "projects" {
  id          = var.id
  name        = var.name
  description = "Project ${var.id}: ${var.name}"
}

resource "unleash_api_token" "tokens" {
  count = length(var.tokens)

  depends_on  = [unleash_project.projects]
  token_name  = "${var.id}-${var.tokens[count.index].environment}-${var.tokens[count.index].type}"
  type        = var.tokens[count.index].type
  expires_at  = "2024-12-31T23:59:59Z"
  project     = var.id
  environment = var.tokens[count.index].environment
}

data "unleash_role" "project_owner_role" {
  name = "Owner"
}

data "unleash_role" "project_member_role" {
  name = "Member"
}

resource "unleash_project_access" "access" {
  project = var.id

  depends_on  = [unleash_project.projects]
  roles = [
    {
      role = data.unleash_role.project_owner_role.id
      users = [var.owner]
      groups = []
    },
    {
      role = data.unleash_role.project_member_role.id
      users = var.members
      groups = []
    },
  ]
}