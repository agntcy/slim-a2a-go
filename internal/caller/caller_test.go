// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package caller

import (
	"context"
	"testing"

	slim_rpc "github.com/agntcy/slim-bindings-go/v3/slim_rpc"
)

type fakeRPCContext struct{ sub, iss *string }

func (f fakeRPCContext) PeerSubject() *string { return f.sub }
func (f fakeRPCContext) PeerIssuer() *string  { return f.iss }

func ptr(s string) *string { return &s }

func TestFromRPCContext(t *testing.T) {
	tests := []struct {
		name   string
		rc     fakeRPCContext
		want   Caller
		wantOK bool
	}{
		{name: "no identity", rc: fakeRPCContext{}},
		{name: "empty subject", rc: fakeRPCContext{sub: ptr(""), iss: ptr("https://issuer")}},
		{
			name:   "subject only",
			rc:     fakeRPCContext{sub: ptr("spiffe://example.org/agent")},
			want:   Caller{Subject: "spiffe://example.org/agent"},
			wantOK: true,
		},
		{
			name:   "subject and issuer",
			rc:     fakeRPCContext{sub: ptr("did:key:z6Mk"), iss: ptr("did:key:z6Mk")},
			want:   Caller{Subject: "did:key:z6Mk", Issuer: "did:key:z6Mk"},
			wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := FromRPCContext(tt.rc)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("FromRPCContext() = %+v, %v; want %+v, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestContextRoundTrip(t *testing.T) {
	if _, ok := FromContext(context.Background()); ok {
		t.Fatal("FromContext(background) reported a caller")
	}
	want := Caller{Subject: "sub", Issuer: "iss"}
	got, ok := FromContext(NewContext(context.Background(), want))
	if !ok || got != want {
		t.Fatalf("FromContext(NewContext(c)) = %+v, %v; want %+v, true", got, ok, want)
	}
	// Metadata in the context, which the caller controls, is not an identity.
	md := slim_rpc.WithMetadata(context.Background(), map[string]string{"subject": "spoofed"})
	if c, ok := FromContext(md); ok {
		t.Fatalf("FromContext(metadata only) = %+v, true; want false", c)
	}
}

// recordingServer records registrations; other ServerInterface methods are not
// used by the generated Register functions.
type recordingServer struct {
	slim_rpc.ServerInterface
	unary  map[string]slim_rpc.UnaryUnaryHandler
	stream map[string]slim_rpc.UnaryStreamHandler
}

func (r *recordingServer) RegisterUnaryUnary(service, method string, h slim_rpc.UnaryUnaryHandler) {
	r.unary[service+"/"+method] = h
}

func (r *recordingServer) RegisterUnaryStream(service, method string, h slim_rpc.UnaryStreamHandler) {
	r.stream[service+"/"+method] = h
}

type nopUnary struct{}

func (nopUnary) Handle([]byte, *slim_rpc.Context) ([]byte, error) { return nil, nil }

type nopStream struct{}

func (nopStream) Handle([]byte, *slim_rpc.Context, *slim_rpc.ResponseSink) error { return nil }

func TestServer_ReplacesGeneratedHandlers(t *testing.T) {
	rec := &recordingServer{
		unary:  map[string]slim_rpc.UnaryUnaryHandler{},
		stream: map[string]slim_rpc.UnaryStreamHandler{},
	}
	ours := nopUnary{}
	oursStream := nopStream{}
	s := Server{
		ServerInterface: rec,
		Unary:           map[string]slim_rpc.UnaryUnaryHandler{"Get": ours},
		Stream:          map[string]slim_rpc.UnaryStreamHandler{"Watch": oursStream},
	}
	s.RegisterUnaryUnary("svc", "Get", nil)
	s.RegisterUnaryStream("svc", "Watch", nil)

	if rec.unary["svc/Get"] != ours {
		t.Errorf("unary svc/Get registered %v, want the adapter", rec.unary["svc/Get"])
	}
	if rec.stream["svc/Watch"] != oursStream {
		t.Errorf("stream svc/Watch registered %v, want the adapter", rec.stream["svc/Watch"])
	}
}

func TestServer_PanicsOnUnknownMethod(t *testing.T) {
	s := Server{ServerInterface: &recordingServer{}}
	for name, register := range map[string]func(){
		"unary":  func() { s.RegisterUnaryUnary("svc", "New", nil) },
		"stream": func() { s.RegisterUnaryStream("svc", "New", nil) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("registering a method with no adapter did not panic")
				}
			}()
			register()
		})
	}
}

func TestWithoutSessionKeys(t *testing.T) {
	md := map[string]string{
		"service":           "svc",
		"method":            "Get",
		"x-app-key":         "v",
		"slim-peer-subject": "agntcy/demo/client_x",
		"slim-payload-sig":  "sig",
	}
	got, changed := withoutSessionKeys(md)
	if !changed {
		t.Fatal("withoutSessionKeys reported no change")
	}
	want := map[string]string{"service": "svc", "method": "Get", "x-app-key": "v"}
	if len(got) != len(want) {
		t.Fatalf("withoutSessionKeys() = %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("withoutSessionKeys()[%q] = %q, want %q", k, got[k], v)
		}
	}
	if _, ok := md["slim-peer-subject"]; !ok {
		t.Error("withoutSessionKeys modified its input")
	}

	clean := map[string]string{"x-app-key": "v"}
	if got, changed := withoutSessionKeys(clean); changed || len(got) != 1 {
		t.Errorf("withoutSessionKeys(clean) = %v, %v; want unchanged", got, changed)
	}
}
