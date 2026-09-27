package create

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// ConnectorResponse is the public successful wire shape. It never includes
// DSN, secret reference, or any other connection material.
type ConnectorResponse struct {
	Name              string          `json:"name"`
	Driver            string          `json:"driver"`
	DSNConfigured     bool            `json:"dsnConfigured"`
	SecretConfigured  bool            `json:"secretConfigured"`
	Description       string          `json:"description,omitempty"`
	OwnerID           string          `json:"ownerId"`
	Status            string          `json:"status"`
	Options           json.RawMessage `json:"options,omitempty"`
	LastTestStatus    string          `json:"lastTestStatus,omitempty"`
	LastTestErrorCode string          `json:"lastTestErrorCode,omitempty"`
	LastTestedAt      *time.Time      `json:"lastTestedAt,omitempty"`
	ETag              int64           `json:"etag"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

func (*ConnectorCreateOutput) JSONWireType() reflect.Type {
	return reflect.TypeFor[ConnectorResponse]()
}

func (output *ConnectorCreateOutput) MarshalJSON() ([]byte, error) {
	if output == nil || output.Data == nil {
		return []byte("null"), nil
	}
	row := output.Data
	if row.Name == "" || row.OwnerId == "" || row.CreatedAt.IsZero() || row.UpdatedAt.IsZero() {
		return nil, fmt.Errorf("created connector response is incomplete")
	}
	result := ConnectorResponse{Name: row.Name, Driver: row.Driver, OwnerID: row.OwnerId,
		Status: row.Status, Options: row.OptionsJson, ETag: row.Etag,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
	if row.DsnTemplate != nil {
		result.DSNConfigured = strings.TrimSpace(*row.DsnTemplate) != ""
	}
	if row.SecretRef != nil {
		result.SecretConfigured = strings.TrimSpace(*row.SecretRef) != ""
	}
	if row.Description != nil {
		result.Description = *row.Description
	}
	return json.Marshal(result)
}
