package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func ValidateApiResponse(response *http.Response, code int, diagnostics *diag.Diagnostics, err error) bool {
	return IsValidApiResponse(response, []int{code}, diagnostics, err)
}

func IsValidApiResponse(response *http.Response, codes []int, diagnostics *diag.Diagnostics, err error) bool {
	if response == nil {
		diagnostics.AddError(
			"Unable to call api, response is nil",
			err.Error())
		return false
	}

	// this is subtle but I've moved this block up because there are cases
	// where 409s are a valid response code but the api will still error
	for _, code := range codes {
		if response.StatusCode == code {
			return true
		}
	}

	if err != nil {
		diagnostics.AddError(
			fmt.Sprintf("Unable to call api %s %s", response.Request.Method, response.Request.URL),
			err.Error(),
		)
	}

	diagnostics.AddError(
		fmt.Sprintf("Unexpected HTTP error code received %s", response.Status),
		fmt.Sprintf("Calling API %s %s\nExpected %v, got %v\n%v", response.Request.Method, response.Request.URL, codes, response.StatusCode, response.Body),
	)

	return false
}

func ValidateReadApiResponse(ctx context.Context, response *http.Response, err error, resp *resource.ReadResponse, resourceId string, resourceName string) bool {
	if response != nil && response.StatusCode == 404 {
		tflog.Warn(ctx, fmt.Sprintf("%s with id %s not found, removing from state", resourceName, resourceId))
		resp.State.RemoveResource(ctx)
		return false
	}

	if !ValidateApiResponse(response, 200, &resp.Diagnostics, err) {
		return false
	}

	return true
}
