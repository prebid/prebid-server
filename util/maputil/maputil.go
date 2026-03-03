package maputil

// ReadEmbeddedMap reads element k from the map m as a map[string]interface{}.
func ReadEmbeddedMap(m map[string]interface{}, k string) (map[string]interface{}, bool) {
	if v, ok := m[k]; ok {
		vCasted, ok := v.(map[string]interface{})
		return vCasted, ok
	}

	return nil, false
}

// ReadEmbeddedSlice reads element k from the map m as a []interface{}.
func ReadEmbeddedSlice(m map[string]interface{}, k string) ([]interface{}, bool) {
	if v, ok := m[k]; ok {
		vCasted, ok := v.([]interface{})
		return vCasted, ok
	}

	return nil, false
}

// ReadEmbeddedString reads element k from the map m as a string.
func ReadEmbeddedString(m map[string]interface{}, k string) (string, bool) {
	if v, ok := m[k]; ok {
		vCasted, ok := v.(string)
		return vCasted, ok
	}
	return "", false
}

// HasElement returns true if nested element k exists.
func HasElement(m map[string]interface{}, k ...string) bool {
	exists := false
	kLastIndex := len(k) - 1

	for i, k := range k {
		isLastKey := i == kLastIndex

		if isLastKey {
			_, exists = m[k]
		} else {
			if m, exists = ReadEmbeddedMap(m, k); !exists {
				break
			}
		}
	}

	return exists
}
