resource "unleash_project" "project_1" {
  id          = "one"
  name        = "First project"
  description = "My first project"
}

resource "unleash_environment" "space" {
  name = "outerspace"
  type = "vacuum"
}

resource "unleash_project_environment" "space_environment" {
  project_id              = unleash_project.project_1.id
  environment_name        = unleash_environment.space.name
  change_requests_enabled = true
  required_approvals      = 2
}