package provider

import (
	"context"
	"fmt"

	"github.com/Azure/alzlib/to"
	"github.com/Azure/terraform-provider-alz/internal/provider/gen"
	"github.com/Azure/terraform-provider-alz/internal/typehelper/gotype"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = (*metadataDataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*metadataDataSource)(nil)

func NewMetadataDataSource() datasource.DataSource {
	return &metadataDataSource{}
}

type metadataDataSource struct {
	alz *alzProviderData
}

func (d *metadataDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_metadata"
}

func (d *metadataDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = gen.MetadataDataSourceSchema(ctx)
}

func (d *metadataDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*alzProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"metadataDataSource.Configure() Unexpected type",
			fmt.Sprintf("Expected *alzProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.alz = data
}

func (d *metadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data gen.MetadataModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.alz == nil {
		resp.Diagnostics.AddError(
			"metadataDataSource.Read() Provider not configured",
			"The provider has not been configured. Please see the provider documentation for configuration instructions.",
		)
		return
	}

	if data.Id.IsNull() || data.Id.IsUnknown() {
		u, err := uuid.GenerateUUID()
		if err != nil {
			resp.Diagnostics.AddError(
				"metadataDataSource.Read() UUID generation failed",
				fmt.Sprintf("Failed to generate UUID: %s", err),
			)
			return
		}
		data.Id = types.StringValue(u)
	}

	alzMeta := d.alz.Metadata()
	alzRefs := make([]string, 0, len(alzMeta))
	for _, ref := range alzMeta {
		if !ref.IsAlzLibraryRef() {
			continue
		}
		alzRefs = append(alzRefs, ref.Ref().String())
	}
	alzRefsAttrVal := gotype.SliceOfPrimitiveToFramework(ctx, to.SliceOfPtrs(alzRefs...))
	data.AlzLibraryReferences = types.ListValueMust(types.StringType, alzRefsAttrVal)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
