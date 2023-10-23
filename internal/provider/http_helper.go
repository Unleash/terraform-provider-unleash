package provider

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func ExpectedResponse(response *http.Response, code int, diagnostics *diag.Diagnostics, err error) bool {
	if response == nil {
		diagnostics.AddError(
			"Unable to call api, response is nil",
			err.Error())
		return false
	}

	if err != nil {
		diagnostics.AddError(
			fmt.Sprintf("Unable to call api %s %s", response.Request.Method, response.Request.URL),
			err.Error(),
		)
	}

	if response.StatusCode != code {
		diagnostics.AddError(
			fmt.Sprintf("Unexpected HTTP error code received %s", response.Status),
			fmt.Sprintf("Calling API %s %s\nExpected %v, got %v\n%v", response.Request.Method, response.Request.URL, code, response.StatusCode, response.Body),
		)
	}

	return !diagnostics.HasError()
}
