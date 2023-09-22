data "unleash_permission" "create_project" {
  name = "CREATE_PROJECT"
}

data "unleash_permission" "update_project" {
  name = "UPDATE_PROJECT"
}

data "unleash_permission" "create_feature_strategy" {
  name        = "CREATE_FEATURE_STRATEGY"
  environment = "development"
}