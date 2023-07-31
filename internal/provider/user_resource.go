package provider

import (
	"context"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

// NewUserResource is a helper function to simplify the provider implementation.
func NewUserResource() resource.Resource {
	return &userResource{}
}

// userResource is the data source implementation.
type userResource struct {
	client *unleash.APIClient
}

type userResourceModel struct {
	Username  types.String  `tfsdk:"username"`
	Email     types.String  `tfsdk:"email"`
	Name      types.String  `tfsdk:"name"`
	Password  types.String  `tfsdk:"password"`
	RootRole  types.Float64 `tfsdk:"rootRole"`
	SendEmail types.Bool    `tfsdk:"sendEmail"`
}

// Configure adds the provider configured client to the data source.
func (d *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unleash.APIClient)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	d.client = client

}

// Metadata returns the data source type name.
func (d *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the schema for the data source. TODO: can we transform OpenAPI schema into TF schema?
func (d *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "User schema",
		Attributes: map[string]schema.Attribute{
			"username": schema.StringAttribute{
				Description: "The username.",
				Optional:    true,
			},
			"email": schema.StringAttribute{
				Description: "The email of the user.",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the user.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "The password of the user.",
				Optional:    true,
			},
			"root_role": schema.Int64Attribute{
				Description: "The role id for the user.",
				Required:    true,
			},
			"send_email": schema.BoolAttribute{
				Description: "Send a welcome email to the customer or not. Defaults to true",
				Optional:    true,
			},
		},
	}
}

func (d *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import user resource")
	var state userResourceModel

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished importing user data source", map[string]any{"success": true})
}

func (d *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to import user resource")
	var state userResourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Debug(ctx, "Finished importing user data source", map[string]any{"success": true})
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
