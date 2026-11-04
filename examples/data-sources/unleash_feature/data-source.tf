data "unleash_feature" "maintenance_mode" {
  name    = "ops.maintenance-mode"
  project = "default"
}

# Use the flag's state in production to decide whether to provision something
resource "null_resource" "maintenance_page" {
  count = data.unleash_feature.maintenance_mode.environments["production"] ? 1 : 0
}
