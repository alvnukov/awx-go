package playbook

import "encoding/json"

func JSONUnmarshalString[T any](objToMapping T) string {
	bytes, _ := json.Marshal(objToMapping)
	return string(bytes)
}
