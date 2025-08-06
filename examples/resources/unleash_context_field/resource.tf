resource "unleash_context_field" "ham_context_field" {
  name = "ham"
}

resource "unleash_context_field" "cheese_context_field" {
  name        = "cheese"
  stickiness  = true
  description = "Type of cheese to constrain on"
  legal_values = [
    {
      value       = "brie"
      description = "Gooey, delicious"
    }
  ]
}