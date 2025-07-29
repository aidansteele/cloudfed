package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-lambda-go/lambdaurl"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"math/big"
	"net/http"
	"os"
)

func main() {
	ctx := context.Background()

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic(err)
	}

	api := kms.NewFromConfig(cfg)

	keyId := os.Getenv("KEY_ID")
	getPub, err := api.GetPublicKey(ctx, &kms.GetPublicKeyInput{KeyId: &keyId})
	if err != nil {
		panic(err)
	}

	pub, err := x509.ParsePKIXPublicKey(getPub.PublicKey)
	if err != nil {
		panic(err)
	}

	srv := Server{
		keyId:  keyId,
		pubKey: pub.(*rsa.PublicKey),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/{tenant}/.well-known/jwks", srv.handleJwks)
	mux.HandleFunc("/{tenant}/.well-known/openid-configuration", srv.handleDiscoveryDocument)
	lambdaurl.Start(mux)
}

type Server struct {
	keyId  string
	pubKey *rsa.PublicKey
}

type oidcDiscoveryDocument struct {
	Issuer                           string   `json:"issuer"`
	JwksUri                          string   `json:"jwks_uri"`
	SubjectTypesSupported            []string `json:"subject_types_supported"`
	ResponseTypesSupported           []string `json:"response_types_supported"`
	IdTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`
	ScopesSupported                  []string `json:"scopes_supported"`
}

type oidcJwk struct {
	Alg string `json:"alg"`
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type oidcJwksResponse struct {
	Keys []oidcJwk `json:"keys"`
}

func (srv *Server) handleJwks(w http.ResponseWriter, r *http.Request) {
	bigE := big.NewInt(int64(srv.pubKey.E))

	response := oidcJwksResponse{
		Keys: []oidcJwk{
			{
				Alg: fmt.Sprintf("RS%d", srv.pubKey.Size()),
				Kty: "RSA",
				Use: "sig",
				Kid: srv.keyId,
				N:   base64.RawURLEncoding.EncodeToString(srv.pubKey.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(bigE.Bytes()),
			},
		},
	}

	j, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

func (srv *Server) handleDiscoveryDocument(w http.ResponseWriter, r *http.Request) {
	tenant := r.PathValue("tenant")

	response := oidcDiscoveryDocument{
		Issuer:                           fmt.Sprintf("https://%s/%s", r.Host, tenant),
		JwksUri:                          fmt.Sprintf("https://%s/%s/.well-known/jwks", r.Host, tenant),
		SubjectTypesSupported:            []string{"public"},
		ResponseTypesSupported:           []string{"id_token"},
		IdTokenSigningAlgValuesSupported: []string{"RS256"},
		ScopesSupported:                  []string{"openid"},
	}

	j, _ := json.Marshal(response)
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}
