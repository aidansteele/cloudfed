package main

import (
	"context"
	"fmt"
	"github.com/aidansteele/cloudfed"
	"github.com/aidansteele/cloudfed/oidc"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	AwsRegion = "us-east-1"
)

func main() {
	ctx := context.Background()

	provider := stscreds.NewWebIdentityRoleProvider(
		sts.NewFromConfig(aws.Config{Region: AwsRegion}),
		cloudfed.AwsRoleArn,
		identityTokenFunc(func() ([]byte, error) {
			token, _, err := oidc.GenerateOidcToken(map[string]any{
				"sub": "example-sub",
				"aud": "sts.amazonaws.com",
			})

			return []byte(token), err
		}),
	)
	cfg, err := config.LoadDefaultConfig(ctx, config.WithCredentialsProvider(provider), config.WithRegion(AwsRegion))
	if err != nil {
		panic(err)
	}

	p := s3.NewListBucketsPaginator(s3.NewFromConfig(cfg), &s3.ListBucketsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			panic(err)
		}

		for _, b := range page.Buckets {
			fmt.Println(*b.Name)
		}
	}
}

type identityTokenFunc func() ([]byte, error)

func (itf identityTokenFunc) GetIdentityToken() ([]byte, error) {
	return itf()
}
