import {
  id = "default"
  to = unleash_project.default_project
}

resource "unleash_project" "default_project" {
  id = "default"
  name = "Taken over by Terraform"
  description = "This was default project"
}
