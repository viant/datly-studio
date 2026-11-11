package value

import (
	"bytes"
	"encoding/json"

	contract "github.com/viant/forge/backend/artifactrelease"
)

// EncodeAudience preserves the legacy private policy array byte shape. Explicit
// public selection uses the same JSON column without manufacturing a policy.
func EncodeAudience(channel contract.Channel) ([]byte, error) {
	if err := contract.ValidateChannel(channel); err != nil {
		return nil, err
	}
	if channel.Public {
		return []byte(`{"public":true}`), nil
	}
	return json.Marshal(channel.Policies)
}

func DecodeAudience(raw []byte, channel *contract.Channel) error {
	if channel == nil {
		return contract.ErrInvalid
	}
	raw = bytes.TrimSpace(raw)
	channel.Public = false
	channel.Policies = nil
	if len(raw) > 0 && raw[0] == '{' {
		var audience struct {
			Public bool `json:"public"`
		}
		if StrictJSON(raw, &audience) != nil || !audience.Public {
			return contract.ErrInvalid
		}
		channel.Public = true
	} else if StrictJSON(raw, &channel.Policies) != nil {
		return contract.ErrInvalid
	}
	return contract.ValidateChannel(*channel)
}
