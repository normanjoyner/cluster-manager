package resources

// syncables maps a syncable to the function that should be called on a cache mismatch
var syncables = make(map[Syncable]func())

// Register adds a syncable for synchronization with Containership Cloud
func Register(s Syncable, onCacheMismatch func()) {
	syncables[s] = onCacheMismatch
}

// ReconcileAll calls Reconcile() on all registered syncables
func ReconcileAll() {
	for s := range syncables {
		s.Reconcile()
	}
}

// ReconcileByType calls Reconcile() on a specific ResourceType
func ReconcileByType(resourceType ResourceType) {
	for s := range syncables {
		if s.GetType() == resourceType {
			s.Reconcile()
		}
	}
}

// Sync calls Sync() on registered syncables
func Sync() error {
	for s, f := range syncables {
		s.Sync(f)
	}

	// TODO
	return nil
}
