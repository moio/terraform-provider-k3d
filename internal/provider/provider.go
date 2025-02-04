package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/k3d-io/k3d/v5/pkg/runtimes"
	"github.com/k3d-io/k3d/v5/pkg/types/fixes"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			Schema: map[string]*schema.Schema{
				"fixes": {
					// https://pkg.go.dev/github.com/k3d-io/k3d/v5/pkg/types/fixes#pkg-variables
					Description: "Explicitly enable or disable K3d fixes during cluster creation.",
					Type:        schema.TypeMap,
					ForceNew:    true,
					Optional:    true,
					Elem:        &schema.Schema{Type: schema.TypeBool},
					ValidateDiagFunc: func(m interface{}, path cty.Path) diag.Diagnostics {
						var diags diag.Diagnostics
						for key := range m.(map[string]interface{}) {
							if _, ok := supportedFixes[key]; !ok {
								diags = append(diags, diag.Diagnostic{
									Severity:      diag.Error,
									Summary:       "Invalid map key",
									Detail:        fmt.Sprintf("Unsupported fix: %s", key),
									AttributePath: append(path, cty.IndexStep{Key: cty.StringVal(key)}),
								})
							}
						}
						return diags
					},
				},
			},
			DataSourcesMap: map[string]*schema.Resource{
				"k3d_cluster":  dataSourceCluster(),
				"k3d_node":     dataSourceNode(),
				"k3d_registry": dataSourceRegistry(),
			},
			ResourcesMap: map[string]*schema.Resource{
				"k3d_cluster":  resourceCluster(),
				"k3d_node":     resourceNode(),
				"k3d_registry": resourceRegistry(),
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

type apiClient struct {
	// Add whatever fields, client or connection info, etc. here
	// you would need to setup to communicate with the upstream
	// API.
}

func configure(version string, p *schema.Provider) func(context.Context, *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		// Setup a User-Agent for your API client (replace the provider name for yours):
		// userAgent := p.UserAgent("terraform-provider-k3d", version)
		// TODO: myClient.UserAgent = userAgent

		if l, ok := d.GetOk("fixes"); ok {
			configureFixes(l.(map[string]interface{}))
		}

		return &apiClient{}, nil
	}
}

// Synced with https://pkg.go.dev/github.com/k3d-io/k3d/v5/pkg/types/fixes#K3DFixEnv
// as of 5.8.1
var supportedFixes = map[string]fixes.K3DFixEnv{
	"cgroupv2": fixes.EnvFixCgroupV2,
	"dns":      fixes.EnvFixDNS,
	"mounts":   fixes.EnvFixMounts,
}

func configureFixes(fixesConfig map[string]interface{}) {
	// Fixes configuration sadly uses a package-level variable, so it can only be configured at provider level and not per cluster
	k3dFixes, _ := fixes.GetFixes(runtimes.SelectedRuntime)
	for name, fix := range supportedFixes {
		v, ok := fixesConfig[name]
		if !ok {
			continue
		}
		k3dFixes[fix] = v.(bool)
	}
}
