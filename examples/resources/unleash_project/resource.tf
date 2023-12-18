import {
  id = "default"
  to = unleash_project.default_project
}

resource "unleash_project" "default_project" {
  id          = "default"
  name        = "Default project"
  description = "Default project now managed by Terraform"
}

resource "unleash_project" "test_project" {
  id          = "my_project"
  name        = "My Terraform project"
  description = "A project created through terraform"
}
