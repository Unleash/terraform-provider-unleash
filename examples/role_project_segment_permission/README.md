# Project role permission update repro

This example validates that a custom project role can be updated to include
the `UPDATE_PROJECT_SEGMENT` project permission.

It intentionally starts with a single permission so the Terraform plan shows
the role being updated in-place when `UPDATE_PROJECT_SEGMENT` is added.

## Prerequisites

Set the Unleash URL and admin token in the environment:

```shell
export UNLEASH_URL="https://sandbox.getunleash.io/enterprise"
export AUTH_TOKEN="<admin-token-or-personal-access-token>"
```

To test a specific provider version, edit `versions.tf` and set the version
constraint, for example:

```hcl
version = "3.2.0"
```

or:

```hcl
version = "3.4.0"
```

## Repro steps

Create the role with the baseline permission:

```shell
terraform init
terraform apply -auto-approve
```

Then enable the `UPDATE_PROJECT_SEGMENT` permission and apply again:

```shell
terraform apply -auto-approve -var include_project_segment_permission=true
terraform plan -detailed-exitcode
```

The second apply should update the role in-place. The final plan should report
`No changes`.

You can also verify through the API:

```shell
ROLE_ID="$(terraform output -raw role_id)"

curl -fsSL \
  -H "Authorization: ${AUTH_TOKEN}" \
  -H "Accept: application/json" \
  "${UNLEASH_URL}/api/admin/roles/${ROLE_ID}" \
  | jq '.permissions[] | select(.name == "UPDATE_PROJECT_SEGMENT")'
```

Clean up the temporary role:

```shell
terraform destroy -auto-approve
```
