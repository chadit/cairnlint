package noexporttest // want `export_test\.go not allowed`

// ExportedForTest exposes an internal for testing. Don't do this.
var ExportedForTest = Placeholder
