package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AlzProviderModel describes the provider data model.
type AlzProviderModel struct {
	AlzLibraryReferences      types.List   `tfsdk:"alz_library_references"`
	AuxiliaryTenantIds        types.List   `tfsdk:"auxiliary_tenant_ids"`
	ClientCertificatePassword types.String `tfsdk:"client_certificate_password"`
	ClientCertificatePath     types.String `tfsdk:"client_certificate_path"`
	ClientId                  types.String `tfsdk:"client_id"`
	ClientSecret              types.String `tfsdk:"client_secret"`
	Environment               types.String `tfsdk:"environment"`
	LibOverwriteEnabled       types.Bool   `tfsdk:"lib_overwrite_enabled"`
	OidcRequestToken          types.String `tfsdk:"oidc_request_token"`
	OidcRequestUrl            types.String `tfsdk:"oidc_request_url"`
	OidcToken                 types.String `tfsdk:"oidc_token"`
	OidcTokenFilePath         types.String `tfsdk:"oidc_token_file_path"`
	SkipProviderRegistration  types.Bool   `tfsdk:"skip_provider_registration"`
	TenantId                  types.String `tfsdk:"tenant_id"`
	UseCli                    types.Bool   `tfsdk:"use_cli"`
	UseMsi                    types.Bool   `tfsdk:"use_msi"`
	UseOidc                   types.Bool   `tfsdk:"use_oidc"`
}

// alzProviderModelGo describes the provider data model as Go types.
type alzProviderModelGo struct {
	alzLibraryReferences      []*AlzProviderModelLibraryReferencesGo
	auxiliaryTenantIds        []*string
	clientCertificatePassword *string
	clientCertificatePath     *string
	clientId                  *string
	clientSecret              *string
	environment               *string
	libOverwriteEnabled       *bool
	oidcRequestToken          *string
	oidcRequestUrl            *string
	oidcToken                 *string
	oidcTokenFilePath         *string
	skipProviderRegistration  *bool
	tenantId                  *string
	useCli                    *bool
	useMsi                    *bool
	useOidc                   *bool
}

type AlzProviderModelLibraryReferences struct {
	Path types.String `tfsdk:"path"`
	Tag  types.String `tfsdk:"tag"`
}

type AlzProviderModelLibraryReferencesGo struct {
	Path *string
	Tag  *string
}

func (m *AlzProviderModel) ToGo(ctx context.Context) *alzProviderModelGo {
	res := new(alzProviderModelGo)
	res.alzLibraryReferences = make([]*AlzProviderModelLibraryReferencesGo, len(m.AlzLibraryReferences.Elements()))
	return nil
}
