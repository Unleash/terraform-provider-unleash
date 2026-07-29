import {
  id = "outerspace"
  to = unleash_environment.space
}

resource "unleash_environment" "space" {
  name = "outerspace"
  type = "vacuum"
}
