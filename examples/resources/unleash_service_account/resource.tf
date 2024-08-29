data "unleash_role" "admin_role" {
  name = "Admin"
}

resource "unleash_service_account" "admin service account" {
  name      = "something unique"
  username  = "something unique"
  root_role = admin_role.id
}
