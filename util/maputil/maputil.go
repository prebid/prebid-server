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

// ReadEmbeddedString reads element k from the map m as a string
func ReadEmbeddedString(m map[string]interface{}, k string) (string, bool) {
	if v, ok := m[k]; ok {
		vCasted, ok := v.(string)
		return vCasted, ok
	}
	return "", false
}
