// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package slimconfig

// The slim-bindings-go v2.1.0 darwin LDFLAGS resolve CoreFoundation symbols
// with -undefined dynamic_lookup but do not link the framework. Programs that
// import net/http get it through crypto/x509; this test binary does not, so
// import crypto/x509 here to pull CoreFoundation into the link on macOS.
import _ "crypto/x509"
