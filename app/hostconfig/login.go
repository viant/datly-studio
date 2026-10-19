package hostconfig

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/viant/scy"
	"github.com/viant/scy/cred"
	"golang.org/x/oauth2"
)

// Resolve loads backend-only OAuth material. Never log or serialize its result.
func (c *Login) Resolve(ctx context.Context) (*oauth2.Config, error) {
	if c == nil {
		return nil, nil
	}
	if c.ClientConfig != nil {
		if c.AuthorizationURL != "" || c.TokenURL != "" || c.ClientID != "" || c.ClientSecret != nil {
			return nil, errors.New("Login.ClientConfig cannot be combined with individual OAuth settings")
		}
		resource := *c.ClientConfig
		resource.SetTarget(reflect.TypeOf(cred.Oauth2Config{}))
		loaded, err := scy.New().Load(ctx, &resource)
		if err != nil {
			return nil, errors.New("Studio OAuth client configuration could not be loaded")
		}
		client, ok := loaded.Target.(*cred.Oauth2Config)
		if !ok || client == nil {
			return nil, errors.New("Studio OAuth client configuration is invalid")
		}
		result := client.Config
		if len(c.Scopes) > 0 {
			result.Scopes = append([]string(nil), c.Scopes...)
		}
		return &result, nil
	}
	result := &oauth2.Config{ClientID: c.ClientID, Endpoint: oauth2.Endpoint{AuthURL: c.AuthorizationURL, TokenURL: c.TokenURL}, Scopes: append([]string(nil), c.Scopes...)}
	if c.ClientSecret != nil {
		value, err := scy.New().Load(ctx, c.ClientSecret)
		if err != nil {
			return nil, errors.New("Studio OAuth client secret could not be loaded")
		}
		result.ClientSecret = strings.TrimSpace(value.String())
	}
	return result, nil
}
