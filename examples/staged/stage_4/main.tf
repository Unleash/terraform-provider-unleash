locals {
  users = {
    justin  = "Justin Time"
    ivan    = "Ivan Issue"
    winston = "Winston Golang"
  }

  projects = {
    first = {
      name = "NullPointer's Paradise",
      tokens = [
        {
          environment = "development"
          type        = "client"
        },
        {
          environment = "development"
          type        = "frontend"
        }
      ]
      owner = unleash_user.users["justin"].id
      users = ["justin", "ivan"]
    }
    third = {
      name = "NullPointer's Paradise 2",
      tokens = [
        {
          environment = "development"
          type        = "client"
        },
        {
          environment = "development"
          type        = "frontend"
        }
      ]
      owner = unleash_user.users["justin"].id
      users = ["justin", "ivan"]
    }
    second = {
      name = "BranchingOut",
      tokens = [
        {
          environment = "development"
          type        = "client"
        },
        {
          environment = "production"
          type        = "client"
        },
      ]
      owner = unleash_user.users["justin"].id
      users = ["ivan", "winston"]
    }
  }
}

module "project" {
  for_each = local.projects

  source  = "./project"
  id      = each.key
  name    = each.value.name
  tokens  = each.value.tokens
  members = [for u in each.value.users : unleash_user.users[u].id]
  owner   = each.value.owner
}

resource "unleash_role" "gatekeeper_role" {
  name        = "Gatekeeper"
  type        = "root-custom"
  description = "This role can create and manage API keys"
  permissions = [
    { "name" : "CREATE_CLIENT_API_TOKEN" },
    { "name" : "UPDATE_CLIENT_API_TOKEN" },
    { "name" : "DELETE_CLIENT_API_TOKEN" },
    { "name" : "CREATE_FRONTEND_API_TOKEN" },
    { "name" : "UPDATE_FRONTEND_API_TOKEN" },
    { "name" : "DELETE_FRONTEND_API_TOKEN" }
  ]
}

resource "unleash_role" "tag_master" {
  name        = "Tag master"
  type        = "root-custom"
  description = "This roles gives the ability to create and manage tags"
  permissions = [
    { "name" : "UPDATE_TAG_TYPE" },
    { "name" : "DELETE_TAG_TYPE" }
  ]
}

resource "unleash_role" "project_manager" {
  name        = "Project manager"
  type        = "custom"
  description = "A role that can control features inside a project"
  permissions = [
    { "name" : "CREATE_FEATURE" },
    { "name" : "UPDATE_FEATURE" },
    { "name" : "DELETE_FEATURE" },
    { "name" : "UPDATE_PROJECT" }
  ]
}

resource "unleash_role" "developer" {
  name        = "Developer"
  type        = "custom"
  description = "A developer role"
  permissions = [
    { "name" : "CREATE_FEATURE_STRATEGY",
    "environment" : "development" },
    { "name" : "DELETE_FEATURE_STRATEGY",
    "environment" : "development" },
    { "name" : "UPDATE_FEATURE_STRATEGY",
    "environment" : "development" },
    { "name" : "UPDATE_FEATURE_ENVIRONMENT",
    "environment" : "development" }
  ]
}

data "unleash_role" "viewer" {
  name = "Viewer"
}

resource "unleash_user" "users" {
  for_each = local.users

  email      = "${each.key}@getunleash.io"
  name       = each.value
  root_role  = data.unleash_role.viewer.id
  send_email = false
}

resource "unleash_api_token" "admin_token" {
  token_name  = "admin_token"
  type        = "admin"
  expires_at  = "2024-12-31T23:59:59Z"
  projects    = ["*"]
  environment = "*"
}
