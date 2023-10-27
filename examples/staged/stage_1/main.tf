resource "unleash_project" "project_1" {
  id          = "one"
  name        = "First project"
  description = "My first project"
}

resource "unleash_role" "gatekeeper_role" {
  name        = "Gatekeeper"
  type        = "root-custom"
  description = "The token guardian"
  permissions = [
    { "name" : "CREATE_CLIENT_API_TOKEN" },
    { "name" : "UPDATE_CLIENT_API_TOKEN" },
    { "name" : "DELETE_CLIENT_API_TOKEN" },
    { "name" : "READ_CLIENT_API_TOKEN" },
    { "name" : "CREATE_FRONTEND_API_TOKEN" },
    { "name" : "UPDATE_FRONTEND_API_TOKEN" },
    { "name" : "DELETE_FRONTEND_API_TOKEN" },
    { "name" : "READ_FRONTEND_API_TOKEN" }
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
