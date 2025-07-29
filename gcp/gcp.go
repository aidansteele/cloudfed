package main

import (
	"bytes"
	resourcemanager "cloud.google.com/go/resourcemanager/apiv3"
	"cloud.google.com/go/resourcemanager/apiv3/resourcemanagerpb"
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aidansteele/cloudfed"
	"github.com/aidansteele/cloudfed/oidc"
	"golang.org/x/oauth2"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"io"
	"net/http"
	"time"
)

func main() {
	ts := oauth2.ReuseTokenSource(nil, &gcpTokenSourcer{
		audience:       cloudfed.GcpWifAudience,
		serviceAccount: cloudfed.GcpServiceAccountEmail,
	})

	fmt.Printf("✅ Successfully authenticated to GCP organization ID: %s.\n", cloudfed.GcpOrganizationId)
	err := listProjects(context.TODO(), ts, cloudfed.GcpOrganizationId)
	if err != nil {
		panic(err)
	}
}

type gcpTokenSourcer struct {
	audience       string
	serviceAccount string
}

func (g *gcpTokenSourcer) Token() (*oauth2.Token, error) {
	claims := map[string]interface{}{
		"sub": "example-sub",
		"aud": g.audience,
	}

	token, _, err := oidc.GenerateOidcToken(claims)
	if err != nil {
		return nil, err
	}

	return exchangeOIDCForGCPAccessToken(g.audience, g.serviceAccount, token)
}

func exchangeOIDCForGCPAccessToken(gcpAudience, gcpServiceAccount, oidcToken string) (*oauth2.Token, error) {
	// STEP 1: Exchange OIDC → Federated access token via STS
	stsPayload := map[string]string{
		"subject_token":        oidcToken,
		"audience":             gcpAudience,
		"grant_type":           "urn:ietf:params:oauth:grant-type:token-exchange",
		"requested_token_type": "urn:ietf:params:oauth:token-type:access_token",
		"subject_token_type":   "urn:ietf:params:oauth:token-type:jwt",
		"scope":                "https://www.googleapis.com/auth/cloud-platform",
	}

	stsBody, _ := json.Marshal(stsPayload)
	stsResp, err := http.Post("https://sts.googleapis.com/v1/token", "application/json", bytes.NewReader(stsBody))
	if err != nil {
		return nil, fmt.Errorf("STS request failed: %w", err)
	}
	defer stsResp.Body.Close()

	var stsResult struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(stsResp.Body).Decode(&stsResult); err != nil {
		return nil, fmt.Errorf("STS decode error: %w", err)
	}

	impersonationURL := fmt.Sprintf(
		"https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/%s:generateAccessToken", gcpServiceAccount,
	)

	impersonationPayload := map[string]interface{}{
		"scope": []string{"https://www.googleapis.com/auth/cloud-platform"},
	}
	impBody, _ := json.Marshal(impersonationPayload)

	req, _ := http.NewRequest("POST", impersonationURL, bytes.NewReader(impBody))
	req.Header.Set("Authorization", "Bearer "+stsResult.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	impResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("impersonation request failed: %w", err)
	}
	defer impResp.Body.Close()

	res, err := io.ReadAll(impResp.Body)
	if err != nil {
		return nil, fmt.Errorf("impersonation request body read failed: %w", err)
	}

	var impResult struct {
		AccessToken string    `json:"accessToken"`
		ExpireTime  time.Time `json:"expireTime"`
	}
	if err := json.Unmarshal(res, &impResult); err != nil {
		return nil, fmt.Errorf("impersonation decode error: %w", err)
	}

	return &oauth2.Token{
		AccessToken: impResult.AccessToken,
		TokenType:   "Bearer",
		Expiry:      impResult.ExpireTime,
	}, nil
}

func listProjects(ctx context.Context, ts oauth2.TokenSource, organizationId string) error {
	client, err := resourcemanager.NewProjectsClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return fmt.Errorf("failed to create resource manager client: %w", err)
	}
	defer client.Close()

	req := &resourcemanagerpb.ListProjectsRequest{
		Parent: fmt.Sprintf("organizations/%s", organizationId),
	}
	it := client.ListProjects(ctx, req)

	for {
		proj, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list projects: %w", err)
		}

		fmt.Printf("\n📦 GCP Project: %s (%s)\n", proj.DisplayName, proj.ProjectId)
		err = listBucketsWithAccessToken(ctx, ts, proj.ProjectId)
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func listBucketsWithAccessToken(ctx context.Context, ts oauth2.TokenSource, projectID string) error {
	client, err := storage.NewClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}
	defer client.Close()

	it := client.Buckets(ctx, projectID)
	for {
		bucket, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return fmt.Errorf("list error: %w", err)
		}
		fmt.Printf("  - 🪣 GCP Storage Bucket: %s → %s\n", bucket.Name, bucket.Location)
	}
	return nil
}
