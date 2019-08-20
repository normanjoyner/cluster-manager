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

// StringSliceContains returns true if the passed slice contains the checkFor
// string, else false.
func StringSliceContains(slice []string, checkFor string) bool {
	for _, s := range slice {
		if s == checkFor {
			return true
		}
	}

	return false
}

// RemoveStringFromSlice returns a copy of the passed string slice with all
// instances of the toRemove string removed if it exists.
// The order of the remaining elements will match the order of the original
// slice.
// If a nil slice is passed, nil will be returned.
func RemoveStringFromSlice(slice []string, toRemove string) []string {
	if slice == nil {
		return nil
	}

	result := make([]string, 0)
	for _, s := range slice {
		if s == toRemove {
			continue
		}

		result = append(result, s)
	}

	return result
}

// AddStringToSliceIfMissing returns a copy of the passed string slice with the
// toAdd string added if it did not exist.
// If toAdd already exists in the passed slice, the original slice is returned.
// The new string will be appended to the end of the slice and the order of the
// original elements will match the order of the original slice.
// If a nil slice is passed, a new slice will be created containing only toAdd.
func AddStringToSliceIfMissing(slice []string, toAdd string) []string {
	for _, s := range slice {
		if s == toAdd {
			// String already exists - no need to add it
			return slice
		}
	}

	return append(slice, toAdd)
}
