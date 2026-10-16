// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package caller

// The slim-bindings-go darwin LDFLAGS (still true in v3.3.3) resolve CoreFoundation symbols
// with -undefined dynamic_lookup but do not link the framework. Programs that
// import net/http get it through crypto/x509; this test binary does not, so
// import crypto/x509 here to pull CoreFoundation into the link on macOS.
import _ "crypto/x509"
