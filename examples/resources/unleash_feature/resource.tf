# A long-lived kill switch, created alongside the service that checks it.
# Its strategies and whether it is on in each environment are managed in Unleash.
resource "unleash_feature" "checkout_kill_switch" {
  name        = "checkout.kill-switch"
  project     = "default"
  type        = "kill-switch"
  description = "Turns off checkout when the payment provider is degraded"
}

# Import an existing flag as <project>/<name>, or just <name> for the default project
import {
  id = "default/search.new-ranking"
  to = unleash_feature.new_ranking
}

resource "unleash_feature" "new_ranking" {
  name            = "search.new-ranking"
  type            = "experiment"
  impression_data = true
  # destroying this resource also deletes the flag from the archive, freeing its name
  archive_on_destroy = false
}
