//go:build !full

package formattington

// embeddedFormattersZip is empty in lite builds (those produced without
// -tags full), so the extraction machinery in embedded.go turns into a
// no-op and the bundled formatters are resolved from the repository, the
// FORMATTINGTON_FORMATTERS_DIR environment variable, or PATH.
var embeddedFormattersZip []byte
