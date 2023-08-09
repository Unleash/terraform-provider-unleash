resource "unleash_user" "chuck" {
  email      = "doesnotneedemail@chucknorris.com"
  name       = "Chuck Norris"
  root_role  = 1
  send_email = false
}