import {
  id = 5
  to = unleash_role.project_member_role
}

resource "unleash_role" "project_member_role" {
  permissions = [{
    name = "CREATE_PROJECT"
    }, {
    name = "UPDATE_PROJECT"
    }, {
    name = "DELETE_PROJECT"
  }]
}

resource "unleash_role" "custom_root_role" {
  name        = "A custom role"
  type        = "root-custom"
  description = "A custom test root role"
  permissions = [{
    name = "CREATE_PROJECT"
    }, {
    name = "UPDATE_PROJECT"
  }]
}

resource "unleash_role" "custom_root_role" {
  name        = "Renamed custom role"
  type        = "root-custom"
  description = "A custom test root role"
  permissions = [{
    name = "CREATE_SEGMENT"
    }, {
    name = "UPDATE_SEGMENT"
  }]
}

resource "unleash_role" "project_role" {
  name        = "Custom project role"
  description = "A custom test project role"
  type        = "custom"
  permissions = [{
    name = "CREATE_FEATURE"
    }, {
    name = "DELETE_FEATURE"
    }, {

    name        = "UPDATE_FEATURE_ENVIRONMENT"
    environment = "development"
  }]
}
