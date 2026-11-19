package mocks

import "context"

// MockDownloader mirrors the signature of the interface it stands in for,
// so the analyzer skips this file because it sits in a mocks/ directory.
type MockDownloader struct {
	DownloadFunc func(ctx context.Context, targetImageObjectID, endpointURL, localPath string, datacenterID int32) error
}
