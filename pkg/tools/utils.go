package tools

// StringMapsAreEqual compares two string maps, checking the first map to see if each
// of the keys are present and contains the same values as the second map
func StringMapsAreEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}

	for k, v := range b {
		value, found := a[k]

		if !found {
			return false
		}

		if value != v {
			return false
		}
	}

	return true
}

// StringSlicesAreEqual returns true if slice a and b are equal, else false.
// If a or b is nil, they are equal if both are nil.
// If two slices contain the same elements but in a different order, they are not equal.
func StringSlicesAreEqual(a, b []string) bool {
	if len(a) != len(b) ||
		a == nil && b != nil ||
		b == nil && a != nil {
		return false
	}

	for i, e := range a {
		if b[i] != e {
			return false
		}
	}

	return true
}
