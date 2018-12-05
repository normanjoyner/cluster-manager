package tools

// StringMapsAreEqual compares two string maps, checking the first map to see if each
// of the keys are present and contains the same values as the second map
func StringMapsAreEqual(new, original map[string]string) bool {
	if len(new) != len(original) {
		return false
	}

	for k, v := range original {
		value, found := new[k]

		if !found {
			return false
		}

		if value != v {
			return false
		}
	}

	return true
}
