data "unleash_user" "admin" {
  id = "1"
}

data "unleash_user" "search_by_email" {
  email = "byemail@example.com"
}
