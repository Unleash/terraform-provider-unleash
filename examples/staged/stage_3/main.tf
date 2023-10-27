resource "unleash_project" "project_1" {
  id          = "one"
  name        = "1st project" # standardize name
  description = "Project one" # standardize description
}

resource "unleash_project" "project_2" {
  id          = "two"
  name        = "2nd project"
  description = "Project two"
}

resource "unleash_role" "gatekeeper_role" {
  name        = "gatekeeper"
  type        = "root-custom"
  description = "The token guardian without the ability of reading the keys"
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
  description = "A custom project role"
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

resource "unleash_user" "dev1" {
  email      = "dev1@getunleash.io"
  name       = "dev_1"
  root_role  = "3"
  send_email = false
}

resource "unleash_user" "dev2" {
  email      = "dev2@getunleash.io"
  name       = "dev_2"
  root_role  = "3" # dev2 should have been a viewer
  send_email = false
}

# dev3 is gone

resource "unleash_api_token" "client_token" {
  token_name  = "dev_client_token"
  type        = "client"
  expires_at  = "2024-12-31T23:59:59Z"
  project     = unleash_project.project_1.id
  environment = "development"
}

resource "unleash_api_token" "client_token_prod" {
  token_name  = "prod_client_token"
  type        = "client"
  expires_at  = "2024-12-31T23:59:59Z"
  project     = unleash_project.project_1.id
  environment = "production"
}

resource "unleash_api_token" "frontend_token" {
  token_name  = "dev_frontend_token"
  type        = "frontend"
  expires_at  = "2024-12-31T23:59:59Z"
  projects    = [unleash_project.project_1.id, unleash_project.project_2.id]
  environment = "development"
}

resource "unleash_api_token" "admin_token" {
  token_name  = "admin_token"
  type        = "admin"
  expires_at  = "2024-12-31T23:59:59Z"
  projects    = ["*"]
  environment = "*"
}
