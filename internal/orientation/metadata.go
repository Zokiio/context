package orientation

import (
	"encoding/json"
	"math"
	"sort"
)

// MetadataForOutput converts decoded YAML metadata to values that retain YAML-
// only information in explicit tagged objects while remaining safe to encode
// as JSON.
func MetadataForOutput(metadata map[string]any) map[string]any {
	result := make(map[string]any, len(metadata))
	for key, value := range metadata {
		result[key] = metadataValue(value)
	}
	return result
}

func metadataValue(value any) any {
	switch value := value.(type) {
	case map[string]any:
		return MetadataForOutput(value)
	case map[any]any:
		type entry struct {
			Key   any `json:"key"`
			Value any `json:"value"`
		}
		entries := make([]entry, 0, len(value))
		for key, item := range value {
			entries = append(entries, entry{Key: metadataValue(key), Value: metadataValue(item)})
		}
		sort.Slice(entries, func(i, j int) bool {
			left, _ := json.Marshal(entries[i].Key)
			right, _ := json.Marshal(entries[j].Key)
			return string(left) < string(right)
		})
		return struct {
			YAMLType string  `json:"yamlType"`
			Entries  []entry `json:"entries"`
		}{"mapping", entries}
	case []any:
		items := make([]any, len(value))
		for i, item := range value {
			items[i] = metadataValue(item)
		}
		return items
	case float64:
		text := ""
		if math.IsNaN(value) {
			text = ".nan"
		} else if math.IsInf(value, 1) {
			text = ".inf"
		} else if math.IsInf(value, -1) {
			text = "-.inf"
		}
		if text != "" {
			return struct {
				YAMLType string `json:"yamlType"`
				Value    string `json:"value"`
			}{"float", text}
		}
	}
	return value
}
