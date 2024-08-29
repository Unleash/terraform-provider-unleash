resource "unleash_service_account" "account_for_tokens_test" {
  name      = "the service account name"
  username  = "some descriptive name"
  root_role = 1
}

resource "unleash_service_account_token" "token_for_account_test" {
  service_account_id = unleash_service_account.account_for_tokens_test.id
  description        = "a token for the account"
  expires_at         = "2048-01-01T00:00:00Z"
}

## This is a bare bones example to output the token to a file, this would be better off being sent to
## a secret manager rather
resource "null_resource" "store_token" {
  provisioner "local-exec" {
    command = <<EOT
      #!/bin/bash
      TOKEN_VALUE="${unleash_service_account_token.token_for_account_test.secret}"
      echo "Storing token: $TOKEN_VALUE"
      echo "$TOKEN_VALUE" > /secure/location/service_account_token.txt
    EOT
  }

  triggers = {
    token_created = unleash_service_account_token.token_for_account_test.secret
  }
}

## Some external resource that requires an Unleash token for automation
resource "external_resource" "external_unleash_integration" {
  unleash_api_key = unleash_service_account_token.token_for_account_test.secret
}
