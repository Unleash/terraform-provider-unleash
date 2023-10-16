resource "unleash_project" "sample_project" {
  id   = "sample"
  name = "sample-project"
}

data "unleash_role" "project_owner_role" {
  name = "Owner"
}

data "unleash_role" "project_member_role" {
  name = "Member"
}

resource "unleash_user" "test_user" {
  name       = "tester"
  email      = "test-password@getunleash.io"
  password   = "you-will-never-guess"
  root_role  = "3"
  send_email = false
}

resource "unleash_user" "test_user_2" {
  name       = "tester-2"
  email      = "test-2-password@getunleash.io"
  password   = "you-will-never-guess"
  root_role  = "3"
  send_email = false
}

resource "unleash_project_access" "sample_project_access" {
  project = unleash_project.sample_project.id
  roles = [
    {
      role = data.unleash_role.project_owner_role.id
      users = [
        unleash_user.test_user.id
      ]
      groups = []
    },
    {
      role = data.unleash_role.project_member_role.id
      users = [
        unleash_user.test_user_2.id
      ]
      groups = []
    },
  ]
}