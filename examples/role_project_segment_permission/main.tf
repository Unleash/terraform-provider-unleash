resource "unleash_role" "project_role" {
  name        = "tf-project-segment-permission-repro"
  description = "Temporary role for validating UPDATE_PROJECT_SEGMENT updates"
  type        = "custom"

  permissions = concat(
    [{
      name = "CREATE_FEATURE"
    }],
    var.include_project_segment_permission ? [{
      name = "UPDATE_PROJECT_SEGMENT"
    }] : [],
  )
}
