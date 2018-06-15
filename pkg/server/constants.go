package server

const (
	// TerminateRole is the role a containership jwt token needs to be signed
	// with to be authenticated to make a request to the /terminate route
	TerminateRole = "terminate"
)
