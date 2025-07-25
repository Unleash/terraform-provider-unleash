package provider

import (
	"context"
	"fmt"

	unleash "github.com/Unleash/unleash-server-api-go/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &contextFieldResource{}
	_ resource.ResourceWithConfigure   = &contextFieldResource{}
	_ resource.ResourceWithImportState = &contextFieldResource{}
)

func NewContextFieldResource() resource.Resource {
	return &contextFieldResource{}
}

type contextFieldResource struct {
	client *unleash.APIClient
}

type contextFieldResourceModel struct {
	Name        types.String        `tfsdk:"name"`
	Description types.String        `tfsdk:"description"`
	Stickiness  types.Bool          `tfsdk:"stickiness"`
	LegalValues basetypes.ListValue `tfsdk:"legal_values"`
}

func (r *contextFieldResource) Configure(ctx context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*unleash.APIClient)
	if !ok {
		tflog.Error(ctx, "Unable to prepare client")
		return
	}
	r.client = client
}

func (r *contextFieldResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_context_field"
}

func (r *contextFieldResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetch a context field.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Description: "The name of the context field.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(), // name is effectively the id so changing this means making a few context field
				},
			},
			"description": schema.StringAttribute{
				Description: "A description of the context field.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(), // description is ignored by the api so if the provider asks for an update, you're getting a new field
				},
			},

			"stickiness": schema.BoolAttribute{
				Description: "Whether this field is available for custom stickiness. Defaults to false if not set.",
				Optional:    true,
				Computed:    true, // the api docs say this field can be null, it's wrong. If it's not set then it's forced to false
			},
			"legal_values": schema.ListNestedAttribute{
				Description: "Legal values for this context field. If not set, then any value is available for this context field.",
				Optional:    true,
				Computed:    true, // the api docs say this field can be null, it's wrong. If it's not set then it's forced to an empty list
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"value": schema.StringAttribute{
							Description: "The allowed value.",
							Required:    true,
						},
						"description": schema.StringAttribute{
							Description: "Description of the allowed value.",
							Optional:    true,
						},
					},
				},
			},
		},
	}
}

func (r *contextFieldResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Debug(ctx, "Preparing to import contextField resource")

	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)

	tflog.Debug(ctx, "Finished importing contextField data source", map[string]interface{}{"success": true})
}

func (r *contextFieldResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Preparing to create contextField resource")
	var plan contextFieldResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createContextFieldRequest := *unleash.NewCreateContextFieldSchemaWithDefaults()
	createContextFieldRequest.Name = *plan.Name.ValueStringPointer()

	var contextErr = populateContextField(ctx, &createContextFieldRequest, plan)
	if contextErr != nil {
		resp.Diagnostics.AddError("error populating context field", contextErr.Error())
		return
	}

	var contextField, httpRes, err = r.client.ContextAPI.CreateContextField(ctx).CreateContextFieldSchema(createContextFieldRequest).Execute()
	if !ValidateApiResponse(httpRes, 201, &resp.Diagnostics, err) {
		return
	}

	plan.hydrateFromApi(*contextField)
	resp.State.Set(ctx, &plan)

	tflog.Debug(ctx, "Finished creating contextField resource", map[string]interface{}{"success": true})
}

func (r *contextFieldResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Preparing to read contextField resource")
	var state contextFieldResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	contextField, httpRes, err := r.client.ContextAPI.GetContextField(ctx, state.Name.ValueString()).Execute()
	if !ValidateApiResponse(httpRes, 200, &resp.Diagnostics, err) {
		return
	}

	state.hydrateFromApi(*contextField)
	resp.State.Set(ctx, &state)

	tflog.Debug(ctx, "Finished reading contextField resource", map[string]interface{}{"success": true})
}

func (r *contextFieldResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Preparing to update contextField resource")
	var plan contextFieldResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateContextFieldRequest := *unleash.NewUpdateContextFieldSchemaWithDefaults()

	var contextErr = populateContextField(ctx, &updateContextFieldRequest, plan)
	if contextErr != nil {
		resp.Diagnostics.AddError("error populating context field", contextErr.Error())
		return
	}

	var httpRes, err = r.client.ContextAPI.UpdateContextField(ctx, plan.Name.ValueString()).UpdateContextFieldSchema(updateContextFieldRequest).Execute()
	if !ValidateApiResponse(httpRes, 200, &resp.Diagnostics, err) {
		return
	}

	contextField, httpRes, err := r.client.ContextAPI.GetContextField(ctx, plan.Name.ValueString()).Execute()
	if !ValidateApiResponse(httpRes, 200, &resp.Diagnostics, err) {
		return
	}

	plan.hydrateFromApi(*contextField)
	resp.State.Set(ctx, &plan)

}

func (r *contextFieldResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Preparing to delete contextField resource")
	var state contextFieldResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	httpRes, err := r.client.ContextAPI.DeleteContextField(ctx, state.Name.ValueString()).Execute()
	if !ValidateApiResponse(httpRes, 200, &resp.Diagnostics, err) {
		return
	}

	resp.State.RemoveResource(ctx)
	tflog.Debug(ctx, "Finished deleting contextField resource", map[string]interface{}{"success": true})
}

func (m *contextFieldResourceModel) hydrateFromApi(api unleash.ContextFieldSchema) {
	m.Name = types.StringValue(api.Name)
	if api.Description.IsSet() && api.Description.Get() != nil {
		m.Description = types.StringValue(*api.Description.Get())
	} else {
		m.Description = types.StringNull()
	}

	if api.Stickiness != nil {
		m.Stickiness = types.BoolValue(*api.Stickiness)
	} else {
		m.Stickiness = types.BoolNull()
	}

	var legalValues []attr.Value

	if api.LegalValues != nil {
		legalValueElements := make([]attr.Value, 0, len(api.LegalValues))
		for _, legalValue := range api.LegalValues {
			if legalValue.Description != nil {
				legalValueElements = append(legalValueElements, types.ObjectValueMust(
					map[string]attr.Type{
						"value":       types.StringType,
						"description": types.StringType,
					},
					map[string]attr.Value{
						"value":       types.StringValue(legalValue.Value),
						"description": types.StringValue(*legalValue.Description),
					},
				))
			} else {
				legalValueElements = append(legalValueElements, types.ObjectValueMust(
					map[string]attr.Type{
						"value":       types.StringType,
						"description": types.StringType,
					},
					map[string]attr.Value{
						"value":       types.StringValue(legalValue.Value),
						"description": types.StringNull(),
					},
				))
			}
		}
		legalValues = legalValueElements
	} else {
		legalValues = []attr.Value{}
	}

	m.LegalValues = types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"value":       types.StringType,
				"description": types.StringType,
			},
		},
		legalValues,
	)
}

func populateContextField(ctx context.Context, request ContextFieldSetter, plan contextFieldResourceModel) error {
	if !plan.Description.IsNull() {
		request.SetDescription(*plan.Description.ValueStringPointer())
	}

	if !plan.Stickiness.IsNull() {
		request.SetStickiness(*plan.Stickiness.ValueBoolPointer())
	} else {
		request.SetStickiness(false)
	}

	if !plan.LegalValues.IsNull() && !plan.LegalValues.IsUnknown() {
		legalValuesList, diags := plan.LegalValues.ToListValue(ctx)
		if diags.HasError() {
			return fmt.Errorf("error extracting legal values: %s", diags)
		}

		var legalValues []unleash.LegalValueSchema

		for _, legalValue := range legalValuesList.Elements() {
			objValue, ok := legalValue.(basetypes.ObjectValue)
			if !ok {
				return fmt.Errorf("non-object value in legal_values: %s", diags)
			}

			valueAttr := objValue.Attributes()["value"]
			descriptionAttr := objValue.Attributes()["description"]

			legalValueSchema := unleash.LegalValueSchema{}

			if stringValue, ok := valueAttr.(types.String); ok && !stringValue.IsNull() && !stringValue.IsUnknown() {
				legalValueSchema.Value = stringValue.ValueString()
			}

			if stringValue, ok := descriptionAttr.(types.String); ok && !stringValue.IsNull() && !stringValue.IsUnknown() {
				legalValueSchema.Description = stringValue.ValueStringPointer()
			}

			legalValues = append(legalValues, legalValueSchema)
		}

		request.SetLegalValues(legalValues)
	} else {
		request.SetLegalValues([]unleash.LegalValueSchema{})
	}
	return nil
}

type ContextFieldSetter interface {
	SetDescription(description string)
	SetStickiness(stickiness bool)
	SetLegalValues(values []unleash.LegalValueSchema)
}
