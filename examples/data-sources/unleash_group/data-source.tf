data "unleash_group" "team" {
  name = "Example Team"
}

data "unleash_group" "team_by_id" {
  id = data.unleash_group.team.id
}
