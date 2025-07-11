---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "unleash_api_token Resource - terraform-provider-unleash"
subcategory: ""
description: |-
  ApiToken schema
---

# unleash_api_token (Resource)

ApiToken schema

## Example Usage

```terraform
resource "unleash_api_token" "frontend_token" {
  token_name  = "frontend_token"
  type        = "frontend"
  expires_at  = "2024-12-31T23:59:59Z"
  projects    = ["*"]
  environment = "development"
}

resource "unleash_api_token" "frontend_token" {
  token_name  = "frontend_token"
  type        = "frontend"
  expires_at  = "2024-12-31T23:59:59Z"
  projects    = ["default"]
  environment = "development"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `environment` (String) An environment the token has access to.
- `expires_at` (String) When the token expires
- `project` (String, Deprecated) A project the token belongs to.
- `projects` (Set of String) The list of projects this token has access to. If the token has access to specific projects they will be listed here. If the token has access to all projects it will be represented as `[*]`.
- `token_name` (String) The name of the token.
- `type` (String) The type of the token.

### Read-Only

- `secret` (String, Sensitive) Secret token value.
