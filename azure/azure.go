package main

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/aidansteele/cloudfed"
	"github.com/aidansteele/cloudfed/oidc"
	"os"
)

func main() {
	azureCred, err := azureCredentials(cloudfed.AzureTenantId, cloudfed.AzureClientId)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create Azure client assertion credential: %v\n", err)
		os.Exit(1)
	}

	authAzure(context.TODO(), azureCred)
}

func azureCredentials(tenantId, clientId string) (azcore.TokenCredential, error) {
	getAssertion := func(ctx context.Context) (string, error) {
		token, _, err := oidc.GenerateOidcToken(map[string]any{
			"sub": "example-sub",
			"aud": "api://AzureADTokenExchange",
		})

		return token, err
	}

	defer fmt.Printf("\n✅ Successfully authenticated to Azure tenant: %s\n🔑 Client ID: %s\n", tenantId, clientId)
	return azidentity.NewClientAssertionCredential(tenantId, clientId, getAssertion, nil)
}

func authAzure(ctx context.Context, cred azcore.TokenCredential) {
	subClient, err := armsubscriptions.NewClient(cred, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create Azure subscription client: %v\n", err)
		os.Exit(1)
	}

	pager := subClient.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to get Azure subscriptions page: %v\n", err)
			os.Exit(1)
		}

		for _, sub := range page.Value {
			subID := *sub.SubscriptionID
			fmt.Printf("\n📦 Azure Subscription: %s (%s)\n", *sub.DisplayName, subID)
			listAzureStorageAccounts(ctx, cred, subID)
		}
	}
}

func listAzureStorageAccounts(ctx context.Context, cred azcore.TokenCredential, subscriptionID string) {
	storageClient, err := armstorage.NewAccountsClient(subscriptionID, cred, nil)
	if err != nil {
		fmt.Printf("  ❌ Failed to create Azure storage client: %v\n", err)
		return
	}

	pager := storageClient.NewListPager(nil)
	found := false
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			fmt.Printf("  ❌ Failed to list Azure storage accounts: %v\n", err)
			return
		}
		for _, acct := range page.Value {
			found = true
			name := *acct.Name
			endpoint := "No blob endpoint"
			if acct.Properties != nil && acct.Properties.PrimaryEndpoints != nil && acct.Properties.PrimaryEndpoints.Blob != nil {
				endpoint = *acct.Properties.PrimaryEndpoints.Blob
			}
			fmt.Printf("  - 🪣 Azure Storage Account: %s → %s\n", name, endpoint)
		}
	}

	if !found {
		fmt.Println("  (No storage accounts found)")
	}
}
