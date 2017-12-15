package resources

type syncable interface {
	// Reconcile checks actual resources on server vs cached
	Reconcile()
	// Sync fetches resource from Cloud API, compares against cache
	// takes in function that is executed if states do not match
	Sync(f func())
	// Write updates state to match desired state
	Write()
}

// CSResource defines what each construct needs to contain
type csResource struct {
	Endpoint string
}
