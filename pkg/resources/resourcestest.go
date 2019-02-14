package resources

// strptr is intended to be used to provide a pointer to a
// static string for easily building test resources
func strptr(s string) *string {
	return &s
}

// int32ptr is intended to be used to provide a pointer to
// an int32 for easily building test resources
func int32ptr(i int32) *int32 {
	return &i
}

// float32ptr is intended to be used to provide a pointer to
// an float32 for easily building test resources
func float32ptr(f float32) *float32 {
	return &f
}
