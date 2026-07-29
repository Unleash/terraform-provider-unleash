import {
  id = "1"
  to = unleash_group.team
}

resource "unleash_user" "group_member" {
  email      = "group-member@example.com"
  name       = "Group Member"
  root_role  = 1
  send_email = false
}

resource "unleash_group" "team" {
  name         = "Example Team"
  description  = "A team managed by Terraform"
  mappings_sso = ["ExampleSSOGroup"]
  root_role    = 1
  users        = [tonumber(unleash_user.group_member.id)]
}
