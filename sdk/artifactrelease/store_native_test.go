package artifactrelease_test

import (
	release "github.com/viant/datly-studio/sdk/artifactrelease"
	"github.com/viant/datly-studio/sdk/artifactrelease/nativestore"
)

var _ release.ReleaseStore = (*nativestore.Store)(nil)
