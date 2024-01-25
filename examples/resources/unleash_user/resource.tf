import {
  id = 1
  to = unleash_user.admin
}

resource "unleash_user" "admin" {
  email      = "admin@chucknorris.com"
  name       = "Chuck Norris"
  root_role  = 1
  send_email = false
}

resource "unleash_user" "chuck" {
  email      = "doesnotneedemail@chucknorris.com"
  name       = "Chuck Norris"
  root_role  = 1
  send_email = false
}

resource "unleash_user" "with_password" {
  email      = "visiblepassword@example.com"
  name       = "Iam Transparent"
  root_role  = 1
  send_email = false
  password   = "youcanseeme"
}
