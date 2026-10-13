// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimconfig

import (
	"flag"
	"testing"
	"time"

	slim_bindings "github.com/agntcy/slim-bindings-go/v2"
)

func parse(t *testing.T, args ...string) *Options {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	o := RegisterFlags(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return o
}

func TestDefaults(t *testing.T) {
	t.Setenv("SLIM_SHARED_SECRET", "")
	o := parse(t)
	if o.Identity != IdentitySharedSecret {
		t.Errorf("Identity = %q, want %q", o.Identity, IdentitySharedSecret)
	}
	if o.SharedSecret != DefaultSharedSecret {
		t.Errorf("SharedSecret = %q, want the test default", o.SharedSecret)
	}
	if _, _, err := o.IdentityConfigs(); err == nil {
		t.Error("IdentityConfigs for shared-secret: want error, got nil")
	}

	cfg, err := o.ClientConfig()
	if err != nil {
		t.Fatalf("ClientConfig: %v", err)
	}
	if !cfg.Tls.Insecure {
		t.Error("default node connection should be plain text")
	}
	if cfg.Auth != nil {
		if _, ok := (*cfg.Auth).(slim_bindings.ClientAuthenticationConfigNone); !ok {
			t.Errorf("default node auth = %T, want none", *cfg.Auth)
		}
	}
}

func TestSharedSecretFromEnv(t *testing.T) {
	t.Setenv("SLIM_SHARED_SECRET", "from-env")
	if o := parse(t); o.SharedSecret != "from-env" {
		t.Errorf("SharedSecret = %q, want from-env", o.SharedSecret)
	}
}

func TestJWTIdentity(t *testing.T) {
	o := parse(t,
		"--identity", "jwt",
		"--jwt-key", "/keys/me.pem",
		"--jwt-subject", "did:key:z6MkExample",
		"--jwt-audience", "a2a, slim",
		"--jwt-duration", "10m",
		"--jwt-verify-key", "/keys/peers.jwks",
	)
	provider, verifier, err := o.IdentityConfigs()
	if err != nil {
		t.Fatalf("IdentityConfigs: %v", err)
	}

	p, ok := provider.(slim_bindings.IdentityProviderConfigJwt)
	if !ok {
		t.Fatalf("provider = %T, want IdentityProviderConfigJwt", provider)
	}
	enc, ok := p.Config.Key.(slim_bindings.JwtKeyTypeEncoding)
	if !ok {
		t.Fatalf("provider key = %T, want JwtKeyTypeEncoding", p.Config.Key)
	}
	if enc.Key.Algorithm != slim_bindings.JwtAlgorithmEdDsa || enc.Key.Format != slim_bindings.JwtKeyFormatPem {
		t.Errorf("signing key = %v/%v, want EdDSA/PEM", enc.Key.Algorithm, enc.Key.Format)
	}
	if f, ok := enc.Key.Key.(slim_bindings.JwtKeyDataFile); !ok || f.Path != "/keys/me.pem" {
		t.Errorf("signing key data = %#v, want file /keys/me.pem", enc.Key.Key)
	}
	if p.Config.Subject == nil || *p.Config.Subject != "did:key:z6MkExample" {
		t.Errorf("subject = %v, want did:key:z6MkExample", p.Config.Subject)
	}
	if p.Config.Issuer == nil || *p.Config.Issuer != "did:key:z6MkExample" {
		t.Errorf("issuer = %v, want the subject (self-issued)", p.Config.Issuer)
	}
	if p.Config.Audience == nil || len(*p.Config.Audience) != 2 || (*p.Config.Audience)[1] != "slim" {
		t.Errorf("audience = %v, want [a2a slim]", p.Config.Audience)
	}
	if p.Config.Duration != 10*time.Minute {
		t.Errorf("duration = %v, want 10m", p.Config.Duration)
	}

	v, ok := verifier.(slim_bindings.IdentityVerifierConfigJwt)
	if !ok {
		t.Fatalf("verifier = %T, want IdentityVerifierConfigJwt", verifier)
	}
	dec, ok := v.Config.Key.(slim_bindings.JwtKeyTypeDecoding)
	if !ok {
		t.Fatalf("verifier key = %T, want JwtKeyTypeDecoding", v.Config.Key)
	}
	if dec.Key.Format != slim_bindings.JwtKeyFormatJwks {
		t.Errorf("verifier format = %v, want JWKS", dec.Key.Format)
	}
	if f, ok := dec.Key.Key.(slim_bindings.JwtKeyDataFile); !ok || f.Path != "/keys/peers.jwks" {
		t.Errorf("verifier key data = %#v, want file /keys/peers.jwks", dec.Key.Key)
	}
	if v.Config.Subject != nil {
		t.Errorf("verifier subject = %v, want nil (accept any trusted key)", *v.Config.Subject)
	}
	if v.Config.Issuer != nil {
		t.Errorf("verifier issuer = %v, want nil (accept any issuer)", *v.Config.Issuer)
	}
}

func TestJWTExplicitIssuers(t *testing.T) {
	o := parse(t,
		"--identity", "jwt",
		"--jwt-key", "k.pem",
		"--jwt-issuer", "https://issuer.example.com",
		"--jwt-verify-key", "k.jwks",
		"--jwt-verify-issuer", "https://issuer.example.com",
	)
	provider, verifier, err := o.IdentityConfigs()
	if err != nil {
		t.Fatalf("IdentityConfigs: %v", err)
	}
	p := provider.(slim_bindings.IdentityProviderConfigJwt)
	if p.Config.Issuer == nil || *p.Config.Issuer != "https://issuer.example.com" {
		t.Errorf("issuer = %v, want https://issuer.example.com", p.Config.Issuer)
	}
	if p.Config.Subject != nil {
		t.Errorf("subject = %v, want nil", *p.Config.Subject)
	}
	v := verifier.(slim_bindings.IdentityVerifierConfigJwt)
	if v.Config.Issuer == nil || *v.Config.Issuer != "https://issuer.example.com" {
		t.Errorf("verifier issuer = %v, want https://issuer.example.com", v.Config.Issuer)
	}
}

func TestJWTIdentityErrors(t *testing.T) {
	tests := map[string][]string{
		"missing signing key": {"--identity", "jwt", "--jwt-subject", "s", "--jwt-verify-key", "k.jwks"},
		"missing issuer":      {"--identity", "jwt", "--jwt-key", "k.pem", "--jwt-verify-key", "k.jwks"},
		"missing verify key":  {"--identity", "jwt", "--jwt-subject", "s", "--jwt-key", "k.pem"},
		"bad algorithm":       {"--identity", "jwt", "--jwt-subject", "s", "--jwt-key", "k.pem", "--jwt-verify-key", "k.jwks", "--jwt-algorithm", "HS256"},
		"bad format":          {"--identity", "jwt", "--jwt-subject", "s", "--jwt-key", "k.pem", "--jwt-verify-key", "k.jwks", "--jwt-verify-format", "x509"},
		"unknown identity":    {"--identity", "oidc"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			if _, _, err := parse(t, args...).IdentityConfigs(); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}

func TestSPIREIdentity(t *testing.T) {
	o := parse(t, "--identity", "spire", "--spire-socket", "unix:///tmp/agent.sock", "--spire-audience", "slim")
	provider, verifier, err := o.IdentityConfigs()
	if err != nil {
		t.Fatalf("IdentityConfigs: %v", err)
	}
	p, ok := provider.(slim_bindings.IdentityProviderConfigSpire)
	if !ok {
		t.Fatalf("provider = %T, want IdentityProviderConfigSpire", provider)
	}
	if p.Config.SocketPath == nil || *p.Config.SocketPath != "unix:///tmp/agent.sock" {
		t.Errorf("socket = %v, want unix:///tmp/agent.sock", p.Config.SocketPath)
	}
	if len(p.Config.JwtAudiences) != 1 || p.Config.JwtAudiences[0] != "slim" {
		t.Errorf("audiences = %v, want [slim]", p.Config.JwtAudiences)
	}
	if p.Config.TargetSpiffeId != nil {
		t.Errorf("target SPIFFE ID = %v, want nil", *p.Config.TargetSpiffeId)
	}
	if _, ok := verifier.(slim_bindings.IdentityVerifierConfigSpire); !ok {
		t.Errorf("verifier = %T, want IdentityVerifierConfigSpire", verifier)
	}
}

func TestTLSBearerClientConfig(t *testing.T) {
	o := parse(t,
		"--slim-endpoint", "https://slim.example.com:443",
		"--tls-ca", "/certs/ca.pem",
		"--node-auth", "bearer",
		"--node-token-file", "/run/token",
	)
	cfg, err := o.ClientConfig()
	if err != nil {
		t.Fatalf("ClientConfig: %v", err)
	}
	if cfg.Endpoint != "https://slim.example.com:443" {
		t.Errorf("endpoint = %q", cfg.Endpoint)
	}
	if cfg.Tls.Insecure || cfg.Tls.InsecureSkipVerify {
		t.Error("TLS should be enabled and verified for an https:// endpoint")
	}
	if ca, ok := cfg.Tls.CaSource.(slim_bindings.CaSourceFile); !ok || ca.Path != "/certs/ca.pem" {
		t.Errorf("CA source = %#v, want file /certs/ca.pem", cfg.Tls.CaSource)
	}
	if cfg.Auth == nil {
		t.Fatal("auth = nil, want static JWT")
	}
	a, ok := (*cfg.Auth).(slim_bindings.ClientAuthenticationConfigStaticJwt)
	if !ok || a.Config.TokenFile != "/run/token" {
		t.Errorf("auth = %#v, want static JWT from /run/token", *cfg.Auth)
	}
}

func TestTLSFlagAndJWTNodeAuth(t *testing.T) {
	o := parse(t,
		"--slim-endpoint", "http://slim.example.com:46357",
		"--tls",
		"--tls-cert", "/certs/me.pem", "--tls-key", "/certs/me.key",
		"--node-auth", "jwt",
		"--jwt-key", "/keys/me.pem",
		"--jwt-subject", "did:key:z6MkExample",
	)
	cfg, err := o.ClientConfig()
	if err != nil {
		t.Fatalf("ClientConfig: %v", err)
	}
	if cfg.Tls.Insecure {
		t.Error("--tls should enable TLS")
	}
	if src, ok := cfg.Tls.Source.(slim_bindings.TlsSourceFile); !ok || src.Cert != "/certs/me.pem" || src.Key != "/certs/me.key" {
		t.Errorf("TLS source = %#v, want client cert files", cfg.Tls.Source)
	}
	if cfg.Auth == nil {
		t.Fatal("auth = nil, want JWT")
	}
	if _, ok := (*cfg.Auth).(slim_bindings.ClientAuthenticationConfigJwt); !ok {
		t.Errorf("auth = %T, want ClientAuthenticationConfigJwt", *cfg.Auth)
	}
}

func TestClientConfigErrors(t *testing.T) {
	tests := map[string][]string{
		"tls option without tls": {"--tls-ca", "ca.pem"},
		"cert without key":       {"--tls", "--tls-cert", "me.pem"},
		"bearer without token":   {"--tls", "--node-auth", "bearer"},
		"jwt without key":        {"--tls", "--node-auth", "jwt"},
		"unknown node auth":      {"--tls", "--node-auth", "basic"},
	}
	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := parse(t, args...).ClientConfig(); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}
