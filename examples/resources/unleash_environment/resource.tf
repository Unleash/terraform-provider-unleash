import {
  id = "outerspace"
  to = unleash_environment.space
}

resource "unleash_environment" "space" {
  name = "outerspace"
  type = "vacuum"
}

# Preconfigure environment level change requests: every project using this
# environment inherits the requirement of two approvals per change request.
resource "unleash_environment" "space_station" {
  name               = "space-station"
  type               = "production"
  required_approvals = 2
}
