//go:build full

package formattington

import _ "embed"

// embeddedFormattersZip carries the bundled formatter executables for the
// build platform as a single zip archive. The members are named
// "<tool>/v<version>/<platform>/<executable>", which is the layout the
// program resolves at runtime; the archive itself is produced by
// release.bash, which populates assets/ and compiles with "-tags full".
//
// go:embed cannot execute the embedded bytes in place, so the archive is
// unpacked into a per-user cache directory on first use (see
// embedded.go).
//
//go:embed assets/formatters.zip
var embeddedFormattersZip []byte
