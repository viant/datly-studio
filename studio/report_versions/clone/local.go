package clone

import (
	"context"
	"path/filepath"

	"github.com/viant/datly/bootstrap/connector"
	"github.com/viant/datly/standalone"
	"github.com/viant/datly/standalone/config"
	"github.com/viant/scy"
	"github.com/viant/scy/auth/jwt/verifier"
)

// NewLocalHost supplies the existing native components to a loopback development
// gateway. It starts no listeners and uses only the gateway's ephemeral verifier.
// Production hosts use their configured native Datly control plane instead.
func NewLocalHost(ctx context.Context, root, dsn string, publicKey []byte) (*standalone.Server, error) {
	cfg, err := (config.Loader{}).Load(ctx, filepath.Join(root, "datly.yaml"))
	if err != nil {
		return nil, err
	}
	cfg.BaseDir = root
	cfg.MCP = nil
	cfg.Endpoint.Address = "127.0.0.1:0"
	cfg.Connectors = []connector.Config{{Name: "studio", Driver: "sqlite", DSN: dsn}, {Name: "authz", AliasOf: "studio"}}
	cfg.JWTValidator = &verifier.Config{RSA: []*scy.Resource{{URL: "studio-development-key", Data: publicKey}}}
	cfg.JWTClaims.Issuer = "studio-development"
	cfg.JWTClaims.Audience = "studio-sdk"
	server, err := standalone.New(ctx, standalone.Options{Config: cfg, RequireLinked: true})
	if err != nil {
		return nil, err
	}
	if err = server.Reload(ctx, 1); err != nil {
		_ = server.Shutdown(ctx)
		return nil, err
	}
	return server, nil
}
