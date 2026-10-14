// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package slimconfig holds the command-line options shared by the echo agent
// examples: how the SLIM app proves and verifies identity, and how it connects
// to the SLIM node.
//
// The default (shared secret over plain-text gRPC) only suits a local node.
// With a shared secret, anyone holding the secret can claim any SLIM name, so
// real deployments should use a JWT or SPIRE identity and a TLS connection.
package slimconfig

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	slim_bindings "github.com/agntcy/slim-bindings-go/v3"
)

// Identity modes accepted by --identity.
const (
	IdentitySharedSecret = "shared-secret"
	IdentityJWT          = "jwt"
	IdentitySPIRE        = "spire"
)

// Node authentication modes accepted by --node-auth.
const (
	NodeAuthNone   = "none"
	NodeAuthBearer = "bearer"
	NodeAuthJWT    = "jwt"
)

// DefaultSharedSecret is the secret used when neither --shared-secret nor
// SLIM_SHARED_SECRET is set. It is for local testing only.
const DefaultSharedSecret = "my_shared_secret_for_testing_purposes_only"

// Options configures the SLIM app identity and the node connection.
type Options struct {
	// Identity selects the identity provider/verifier: shared-secret, jwt or spire.
	Identity string
	// SharedSecret is the symmetric secret for the shared-secret identity (test only).
	SharedSecret string

	// JWTKeyFile is the private key used to sign this app's JWTs (PEM).
	JWTKeyFile string
	// JWTAlgorithm is the signing/verification algorithm, e.g. EdDSA, ES256, RS256.
	JWTAlgorithm string
	// JWTSubject is the "sub" claim, e.g. a did:key. Empty lets SLIM pick a default.
	JWTSubject string
	// JWTIssuer is the "iss" claim to set. Empty uses JWTSubject (self-issued).
	JWTIssuer string
	// JWTAudience is a comma-separated list of "aud" values to set and verify.
	JWTAudience string
	// JWTDuration is the validity of each signed token.
	JWTDuration time.Duration
	// JWTVerifyKeyFile holds the public key(s) trusted to verify peers.
	JWTVerifyKeyFile string
	// JWTVerifyFormat is the format of JWTVerifyKeyFile: jwks, jwk or pem.
	JWTVerifyFormat string
	// JWTVerifyIssuer, when set, is the only "iss" accepted from peers.
	JWTVerifyIssuer string

	// SpireSocket is the SPIFFE Workload API socket. Empty uses SPIFFE_ENDPOINT_SOCKET.
	SpireSocket string
	// SpireTargetSpiffeID is the optional SPIFFE ID to request JWT-SVIDs for.
	SpireTargetSpiffeID string
	// SpireAudience is a comma-separated list of JWT-SVID audiences.
	SpireAudience string

	// Endpoint is the SLIM node address.
	Endpoint string
	// TLS enables TLS to the node. An https:// or wss:// endpoint enables it too.
	TLS bool
	// TLSCAFile is a PEM CA bundle for verifying the node certificate.
	TLSCAFile string
	// TLSInsecureSkipVerify disables node certificate verification (testing only).
	TLSInsecureSkipVerify bool
	// TLSCertFile and TLSKeyFile are an optional client certificate for mTLS.
	TLSCertFile string
	TLSKeyFile  string
	// NodeAuth selects how the app authenticates to the node: none, bearer or jwt.
	NodeAuth string
	// NodeTokenFile holds the bearer token for --node-auth bearer. The file is
	// re-read when it changes, so a token refresher can rotate it.
	NodeTokenFile string
}

// RegisterFlags registers the options on fs and returns the struct they fill.
func RegisterFlags(fs *flag.FlagSet) *Options {
	o := &Options{}

	fs.StringVar(&o.Identity, "identity", IdentitySharedSecret,
		"SLIM identity: shared-secret (local testing only), jwt or spire")
	fs.StringVar(&o.SharedSecret, "shared-secret", envOr("SLIM_SHARED_SECRET", DefaultSharedSecret),
		"shared secret for --identity shared-secret (env SLIM_SHARED_SECRET); local testing only")

	fs.StringVar(&o.JWTKeyFile, "jwt-key", "", "PEM private key used to sign this app's JWTs (--identity jwt)")
	fs.StringVar(&o.JWTAlgorithm, "jwt-algorithm", "EdDSA", "JWT algorithm: EdDSA, ES256, ES384, RS256, RS384, RS512, PS256, PS384, PS512")
	fs.StringVar(&o.JWTSubject, "jwt-subject", "", "JWT subject claim, e.g. did:key:z6Mk...")
	fs.StringVar(&o.JWTIssuer, "jwt-issuer", "", "JWT issuer claim to set (default: --jwt-subject)")
	fs.StringVar(&o.JWTAudience, "jwt-audience", "", "comma-separated JWT audiences to set and verify")
	fs.DurationVar(&o.JWTDuration, "jwt-duration", time.Hour, "validity of each signed JWT")
	fs.StringVar(&o.JWTVerifyKeyFile, "jwt-verify-key", "", "file with the public key(s) trusted to verify peers (--identity jwt)")
	fs.StringVar(&o.JWTVerifyFormat, "jwt-verify-format", "jwks", "format of --jwt-verify-key: jwks, jwk or pem")
	fs.StringVar(&o.JWTVerifyIssuer, "jwt-verify-issuer", "", "only accept peer JWTs from this issuer (default: any issuer)")

	fs.StringVar(&o.SpireSocket, "spire-socket", "", "SPIFFE Workload API socket (default: SPIFFE_ENDPOINT_SOCKET)")
	fs.StringVar(&o.SpireTargetSpiffeID, "spire-target-spiffe-id", "", "SPIFFE ID to request JWT-SVIDs for")
	fs.StringVar(&o.SpireAudience, "spire-audience", "", "comma-separated JWT-SVID audiences")

	fs.StringVar(&o.Endpoint, "slim-endpoint", "http://127.0.0.1:46357", "SLIM node endpoint")
	fs.BoolVar(&o.TLS, "tls", false, "connect to the node over TLS (implied by an https:// or wss:// endpoint)")
	fs.StringVar(&o.TLSCAFile, "tls-ca", "", "PEM CA bundle for the node certificate (system roots are also used)")
	fs.BoolVar(&o.TLSInsecureSkipVerify, "tls-insecure-skip-verify", false, "skip node certificate verification; testing only")
	fs.StringVar(&o.TLSCertFile, "tls-cert", "", "client certificate for mTLS to the node")
	fs.StringVar(&o.TLSKeyFile, "tls-key", "", "client private key for mTLS to the node")
	fs.StringVar(&o.NodeAuth, "node-auth", NodeAuthNone, "node authentication: none, bearer (--node-token-file) or jwt (signs with --jwt-*)")
	fs.StringVar(&o.NodeTokenFile, "node-token-file", "", "file holding the bearer token for --node-auth bearer")

	return o
}

// CreateApp creates the SLIM app for name on svc using the configured identity.
func (o *Options) CreateApp(svc *slim_bindings.Service, name *slim_bindings.Name) (*slim_bindings.App, error) {
	if o.Identity == IdentitySharedSecret {
		if o.SharedSecret == "" {
			return nil, errors.New("--shared-secret must not be empty")
		}
		slog.Warn("using a shared-secret SLIM identity: anyone with the secret can claim any name; use --identity jwt or spire outside local testing")
		return svc.CreateAppWithSecret(name, o.SharedSecret)
	}
	provider, verifier, err := o.IdentityConfigs()
	if err != nil {
		return nil, err
	}
	return svc.CreateApp(name, provider, verifier)
}

// Connect connects svc to the configured SLIM node and returns the connection ID.
func (o *Options) Connect(svc *slim_bindings.Service) (uint64, error) {
	cfg, err := o.ClientConfig()
	if err != nil {
		return 0, err
	}
	return svc.Connect(cfg)
}

// IdentityConfigs returns the identity provider and verifier for the jwt and
// spire identities. The shared-secret identity has no configs here: CreateApp
// uses CreateAppWithSecret for it.
func (o *Options) IdentityConfigs() (slim_bindings.IdentityProviderConfig, slim_bindings.IdentityVerifierConfig, error) {
	switch o.Identity {
	case IdentityJWT:
		signer, err := o.jwtSigner()
		if err != nil {
			return nil, nil, err
		}
		if o.JWTVerifyKeyFile == "" {
			return nil, nil, errors.New("--identity jwt requires --jwt-verify-key")
		}
		alg, err := ParseJWTAlgorithm(o.JWTAlgorithm)
		if err != nil {
			return nil, nil, err
		}
		format, err := ParseJWTKeyFormat(o.JWTVerifyFormat)
		if err != nil {
			return nil, nil, err
		}
		// The verifier checks the audience (and the issuer, if pinned) but not
		// the subject: it accepts any peer whose token is signed by a trusted
		// key. With a JWKS file the key set is the allowlist of peers.
		verifier := slim_bindings.IdentityVerifierConfigJwt{Config: slim_bindings.JwtAuth{
			Key: slim_bindings.JwtKeyTypeDecoding{Key: slim_bindings.JwtKeyConfig{
				Algorithm: alg,
				Format:    format,
				Key:       slim_bindings.JwtKeyDataFile{Path: o.JWTVerifyKeyFile},
			}},
			Audience: splitList(o.JWTAudience),
			Issuer:   optional(o.JWTVerifyIssuer),
			Duration: o.JWTDuration,
		}}
		return slim_bindings.IdentityProviderConfigJwt{Config: signer}, verifier, nil

	case IdentitySPIRE:
		audiences := []string{}
		if a := splitList(o.SpireAudience); a != nil {
			audiences = *a
		}
		spire := slim_bindings.SpireConfig{
			SocketPath:     optional(o.SpireSocket),
			TargetSpiffeId: optional(o.SpireTargetSpiffeID),
			JwtAudiences:   audiences,
			TrustDomains:   []string{},
		}
		return slim_bindings.IdentityProviderConfigSpire{Config: spire},
			slim_bindings.IdentityVerifierConfigSpire{Config: spire},
			nil

	case IdentitySharedSecret:
		return nil, nil, errors.New("the shared-secret identity has no provider/verifier configs; use CreateApp")

	default:
		return nil, nil, fmt.Errorf("unknown --identity %q: must be %s, %s or %s",
			o.Identity, IdentitySharedSecret, IdentityJWT, IdentitySPIRE)
	}
}

// ClientConfig returns the node connection config: TLS and authentication.
func (o *Options) ClientConfig() (slim_bindings.ClientConfig, error) {
	useTLS := o.TLS || strings.HasPrefix(o.Endpoint, "https://") || strings.HasPrefix(o.Endpoint, "wss://")

	var cfg slim_bindings.ClientConfig
	if useTLS {
		cfg = slim_bindings.NewSecureClientConfig(o.Endpoint)
		if o.TLSCAFile != "" {
			cfg.Tls.CaSource = slim_bindings.CaSourceFile{Path: o.TLSCAFile}
		}
		cfg.Tls.InsecureSkipVerify = o.TLSInsecureSkipVerify
		switch {
		case o.TLSCertFile != "" && o.TLSKeyFile != "":
			cfg.Tls.Source = slim_bindings.TlsSourceFile{Cert: o.TLSCertFile, Key: o.TLSKeyFile}
		case o.TLSCertFile != "" || o.TLSKeyFile != "":
			return cfg, errors.New("--tls-cert and --tls-key must be set together")
		}
	} else {
		if o.TLSCAFile != "" || o.TLSCertFile != "" || o.TLSKeyFile != "" || o.TLSInsecureSkipVerify {
			return cfg, errors.New("--tls-* options need --tls or an https:// endpoint")
		}
		cfg = slim_bindings.NewInsecureClientConfig(o.Endpoint)
	}

	var auth slim_bindings.ClientAuthenticationConfig
	switch o.NodeAuth {
	case NodeAuthNone, "":
		return cfg, nil
	case NodeAuthBearer:
		if o.NodeTokenFile == "" {
			return cfg, errors.New("--node-auth bearer requires --node-token-file")
		}
		auth = slim_bindings.ClientAuthenticationConfigStaticJwt{Config: slim_bindings.StaticJwtAuth{
			TokenFile: o.NodeTokenFile,
			Duration:  time.Hour,
		}}
	case NodeAuthJWT:
		signer, err := o.jwtSigner()
		if err != nil {
			return cfg, err
		}
		auth = slim_bindings.ClientAuthenticationConfigJwt{Config: signer}
	default:
		return cfg, fmt.Errorf("unknown --node-auth %q: must be %s, %s or %s",
			o.NodeAuth, NodeAuthNone, NodeAuthBearer, NodeAuthJWT)
	}
	if !useTLS {
		slog.Warn("sending node credentials over a plain-text connection; use --tls")
	}
	cfg.Auth = &auth
	return cfg, nil
}

// jwtSigner returns the JWT signing config built from the --jwt-* options.
func (o *Options) jwtSigner() (slim_bindings.ClientJwtAuth, error) {
	if o.JWTKeyFile == "" {
		return slim_bindings.ClientJwtAuth{}, errors.New("JWT signing requires --jwt-key")
	}
	// SLIM verifiers reject tokens without an "iss" claim. A self-issued
	// token (iss = sub, as with did:key) is the natural default.
	issuer := o.JWTIssuer
	if issuer == "" {
		issuer = o.JWTSubject
	}
	if issuer == "" {
		return slim_bindings.ClientJwtAuth{}, errors.New("JWT signing requires --jwt-subject or --jwt-issuer")
	}
	alg, err := ParseJWTAlgorithm(o.JWTAlgorithm)
	if err != nil {
		return slim_bindings.ClientJwtAuth{}, err
	}
	return slim_bindings.ClientJwtAuth{
		Key: slim_bindings.JwtKeyTypeEncoding{Key: slim_bindings.JwtKeyConfig{
			Algorithm: alg,
			Format:    slim_bindings.JwtKeyFormatPem,
			Key:       slim_bindings.JwtKeyDataFile{Path: o.JWTKeyFile},
		}},
		Audience: splitList(o.JWTAudience),
		Issuer:   &issuer,
		Subject:  optional(o.JWTSubject),
		Duration: o.JWTDuration,
	}, nil
}

// ParseJWTAlgorithm maps an algorithm name such as "EdDSA" or "ES256" to its
// slim_bindings value. Matching is case-insensitive. HMAC algorithms are not
// accepted: use the shared-secret identity for symmetric keys.
func ParseJWTAlgorithm(s string) (slim_bindings.JwtAlgorithm, error) {
	switch strings.ToUpper(s) {
	case "EDDSA", "ED25519":
		return slim_bindings.JwtAlgorithmEdDsa, nil
	case "ES256":
		return slim_bindings.JwtAlgorithmEs256, nil
	case "ES384":
		return slim_bindings.JwtAlgorithmEs384, nil
	case "RS256":
		return slim_bindings.JwtAlgorithmRs256, nil
	case "RS384":
		return slim_bindings.JwtAlgorithmRs384, nil
	case "RS512":
		return slim_bindings.JwtAlgorithmRs512, nil
	case "PS256":
		return slim_bindings.JwtAlgorithmPs256, nil
	case "PS384":
		return slim_bindings.JwtAlgorithmPs384, nil
	case "PS512":
		return slim_bindings.JwtAlgorithmPs512, nil
	default:
		return 0, fmt.Errorf("unsupported JWT algorithm %q", s)
	}
}

// ParseJWTKeyFormat maps "jwks", "jwk" or "pem" (case-insensitive) to its
// slim_bindings value.
func ParseJWTKeyFormat(s string) (slim_bindings.JwtKeyFormat, error) {
	switch strings.ToLower(s) {
	case "jwks":
		return slim_bindings.JwtKeyFormatJwks, nil
	case "jwk":
		return slim_bindings.JwtKeyFormatJwk, nil
	case "pem":
		return slim_bindings.JwtKeyFormatPem, nil
	default:
		return 0, fmt.Errorf("unsupported JWT key format %q: must be jwks, jwk or pem", s)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func splitList(s string) *[]string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return &out
}
