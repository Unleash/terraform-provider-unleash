import {
  id = 1
  to = unleash_service_account.admin_service_account
}

data "unleash_role" "admin_role" {
  name = "Admin"
}

resource "unleash_service_account" "admin_service_account" {
  name      = "something unique"
  username  = "something unique"
  root_role = data.unleash_role.admin_role.id
}
