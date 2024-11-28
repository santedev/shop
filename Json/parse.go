package Json

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type jsonParser struct {
	data any // Can hold either pointer of map[string]any or []any
}

func NewJsonParser(data []byte) (jsonParser, error) {
	if len(data) <= 0 {
		return jsonParser{}, fmt.Errorf("input data for Unmarshal is len 0, cant unmarshal data then")
	}
	v := mapForMarshal(data[0])
	err := json.Unmarshal(data, &v)
	if err != nil {
		return jsonParser{}, err
	}
	return jsonParser{data: &v}, nil
}

func (jp *jsonParser) Get(keys ...string) (any, error) {
	var value any = jp.data
	for _, key := range keys {
		switch v := value.(type) {
		case map[string]any:
			if nextValue, ok := v[key]; ok {
				value = nextValue
			} else {
				return nil, fmt.Errorf("could not find value with key: %s", key)
			}
		case []any:
			index, err := strconv.Atoi(key)
			switch {
			case err != nil:
				return nil, err
			case index >= len(v):
				return nil, fmt.Errorf("index is out of bounds, index/key greater or equals than len array. index: %d, len: %d", index, len(v))
			case index < 0:
				return nil, fmt.Errorf("index is out of bounds, index: %d is less than 0", index)
			default:
				value = v[index]
			}
		default:
			return nil, fmt.Errorf("could not find a map or array for the current key: %s", key)
		}
	}
	return value, nil
}

func (jp *jsonParser) GetString(keys ...string) (string, error) {
	value, err := jp.Get(keys...)
	if str, ok := value.(string); ok && err == nil {
		return str, nil
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("string not found, instead has value: %v", value)
}

func mapForMarshal(data byte) any {
	if data == '{' {
		return map[string]any{}
	}
	return []any{}
}
