package connectordisplay

import (
	"strings"
	"testing"
)

func TestDisplaySavedConfiguration(t *testing.T) {
	for _, value := range []string{"bigquery://viant-e2e/adlogs", "$Username:$Password@tcp(db:3306)/ci_ads?parseTime=true", "postgres://user:${Password}@db/reporting", "file:/tmp/catalog.db"} {
		if got := DSN(value); got != value {
			t.Errorf("template changed: got %q want %q", got, value)
		}
	}
	for _, value := range []string{"user:do-not-return@tcp(db:3306)/db", "postgres://user:do-not-return@db/reporting", "host=db password='do-not-return' dbname=reporting", "bigquery://project/data?%70assword=do-not-return", "inlined:do-not-return", "jdbc:oracle:thin:user/do-not-return@host"} {
		got := DSN(value)
		if strings.Contains(got, "do-not-return") {
			t.Errorf("credential exposed for input kind: %q", got)
		}
	}
	for _, value := range []string{"/home/operator/.secret/mysql.enc|blowfish://default", "gs://bucket/secret.json|blowfish://default"} {
		if got := SecretReference(value); got != value {
			t.Errorf("reference changed: %q", got)
		}
	}
	for _, value := range []string{"inlined:do-not-return", "data:do-not-return", "do-not-return"} {
		if got := SecretReference(value); strings.Contains(got, "do-not-return") {
			t.Fatal("secret value exposed")
		}
	}
}
