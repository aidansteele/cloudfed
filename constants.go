package cloudfed

import (
	_ "embed"
	"encoding/json"
)

//go:embed tfoutput.json
var tfoutput []byte

// these values get populated from tfoutput.json, which itself
// is created by the makefile script running `terraform output`.
var (
	KeyId     = ""
	IssuerUrl = ""

	AzureTenantId = ""
	AzureClientId = ""

	GcpOrganizationId      = ""
	GcpWifAudience         = ""
	GcpServiceAccountEmail = ""

	AwsRoleArn = ""
)

func init() {
	tfjson := map[string]tfOutputValue{}
	err := json.Unmarshal(tfoutput, &tfjson)
	if err != nil {
		panic(err)
	}

	KeyId = tfjson["key_id"].Value
	IssuerUrl = tfjson["issuer_url"].Value
	AzureTenantId = tfjson["azure_tenant_id"].Value
	AzureClientId = tfjson["azure_client_id"].Value
	GcpOrganizationId = tfjson["gcp_organization_id"].Value
	GcpWifAudience = tfjson["gcp_audience"].Value
	GcpServiceAccountEmail = tfjson["gcp_service_account"].Value
	AwsRoleArn = tfjson["aws_role_arn"].Value
}

type tfOutputValue struct {
	Value string `json:"value"`
}
