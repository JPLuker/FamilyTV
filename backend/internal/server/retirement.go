package server

import "encoding/json"

// Preserve household records verbatim during migration, including lineup data
// for backup recovery. Only viewers using a retired channel lose their session.
func retireChannels(document []byte) ([]byte, error) {
	var household map[string]json.RawMessage
	if err := json.Unmarshal(document, &household); err != nil {
		return nil, err
	}
	if channels, ok := household["channels"]; ok {
		household["archived_channels"] = channels
		delete(household, "channels")
	}
	if raw, ok := household["viewing"]; ok {
		var sessions map[string]json.RawMessage
		if err := json.Unmarshal(raw, &sessions); err != nil {
			return nil, err
		}
		for id, record := range sessions {
			var old struct {
				Channel string `json:"channel"`
			}
			if err := json.Unmarshal(record, &old); err != nil {
				return nil, err
			}
			if old.Channel != "" {
				delete(sessions, id)
			}
		}
		records, err := json.Marshal(sessions)
		if err != nil {
			return nil, err
		}
		household["viewing"] = records
	}
	return json.Marshal(household)
}
