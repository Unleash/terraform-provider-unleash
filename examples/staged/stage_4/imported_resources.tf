import {
  id = "default"
  to = unleash_project.default_project
}

resource "unleash_project" "default_project" {
  id          = "default"
  name        = "Taken over by Terraform"
  description = "This was default project"
}

import {
  id = 1
  to = unleash_user.admin_user
}

resource "unleash_user" "admin_user" {
  root_role = 1
  username  = "admin"
}

import {
  id = "default"
  to = unleash_project_access.default_project_access
}

data "unleash_role" "project_owner_role" {
  name = "Owner"
}

resource "unleash_project_access" "default_project_access" {
  project = "default"
  roles = [
    {
      role = data.unleash_role.project_owner_role.id
      users = [
        unleash_user.admin_user.id
      ]
      groups = []
    },
  ]
}