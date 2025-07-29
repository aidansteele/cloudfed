package oidc

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/aidansteele/cloudfed"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/golang-jwt/jwt/v5"
	"maps"
	"time"
)

// these values come from the terraform output

func GenerateOidcToken(claims map[string]any) (string, time.Time, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile("ak2-mumululu"),
		config.WithRegion("ap-southeast-2"),
	)
	if err != nil {
		return "", time.Time{}, err
	}

	api := kms.NewFromConfig(cfg)

	expiry := time.Now().Add(time.Hour)

	method := &KmsSigningMethod{client: api}

	c := map[string]interface{}{
		"iss": cloudfed.IssuerUrl,
		"iat": time.Now().Unix(),
		"exp": expiry.Unix(),
	}
	maps.Copy(c, claims)

	outputToken := jwt.NewWithClaims(method, jwt.MapClaims(c))
	outputToken.Header["kid"] = cloudfed.KeyId

	token, err := outputToken.SignedString(nil)
	if err != nil {
		return "", time.Time{}, err
	}

	return token, expiry, nil
}

var _ jwt.SigningMethod = (*KmsSigningMethod)(nil)

type KmsSigningMethod struct {
	client *kms.Client
}

func (k *KmsSigningMethod) Sign(signingString string, key interface{}) ([]byte, error) {
	digest := sha256.Sum256([]byte(signingString))

	sign, err := k.client.Sign(context.TODO(), &kms.SignInput{
		KeyId:            aws.String(cloudfed.KeyId),
		Message:          digest[:],
		SigningAlgorithm: types.SigningAlgorithmSpecRsassaPkcs1V15Sha256,
		MessageType:      types.MessageTypeDigest,
	})
	if err != nil {
		return nil, fmt.Errorf("kms signing: %w", err)
	}

	return sign.Signature, nil
}

func (k *KmsSigningMethod) Alg() string {
	return "RS256"
}

func (k *KmsSigningMethod) Verify(signingString string, sig []byte, key interface{}) error {
	panic("verify not implemented")
}
