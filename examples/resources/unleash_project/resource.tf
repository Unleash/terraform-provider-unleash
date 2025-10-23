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

  mode = "protected"

  feature_naming = {
    pattern     = "^feature_[a-z0-9_-]+$"
    example     = "feature_user_signup"
    description = "Feature keys must start with feature_ and use lowercase alphanumerics."
  }

  link_templates = [
    {
      title        = "Product Spec"
      url_template = "https://docs.example.com/projects/{{project}}/features/{{feature}}"
    },
    {
      title        = "Issue Tracker"
      url_template = "https://issues.example.com/browse/{{feature}}"
    }
  ]
}
