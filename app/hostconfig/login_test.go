package hostconfig

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/viant/scy"
	"github.com/viant/scy/cred"
	"golang.org/x/oauth2"
)

func TestLoginResolvesWholeSCYClient(t *testing.T) {
	want := oauth2.Config{ClientID: "synthetic-client", ClientSecret: "synthetic-private", Endpoint: oauth2.Endpoint{AuthURL: "https://identity.invalid/authorize", TokenURL: "https://identity.invalid/token"}, Scopes: []string{"openid"}}
	encoded, err := json.Marshal(cred.Oauth2Config{Config: want})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "client.json")
	if err = os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	resource := &scy.Resource{URL: path, Key: "blowfish://default"}
	got, err := (&Login{ClientConfig: resource}).Resolve(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatal("resolved client differs")
	}
	if resource.URL != path {
		t.Fatal("source configuration mutated")
	}
}
func TestLoginRejectsAmbiguousAndUnreadableClient(t *testing.T) {
	for _, login := range []*Login{
		{ClientConfig: &scy.Resource{URL: "unreadable"}, ClientID: "ambiguous"},
		{ClientConfig: &scy.Resource{URL: filepath.Join(t.TempDir(), "sensitive-client-location")}},
	} {
		_, err := login.Resolve(context.Background())
		if err == nil {
			t.Fatal("expected rejection")
		}
		if strings.Contains(err.Error(), "sensitive-client-location") {
			t.Fatal("secret location leaked")
		}
	}
}
