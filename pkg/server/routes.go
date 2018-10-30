package server

import (
	"net/http"

	"github.com/containership/cluster-manager/pkg/server/handlers"
)

// initializeRoutes sets up all routes
func (s *CSServer) initializeRoutes() {
	m := &handlers.Metadata{}
	c := &handlers.Terminate{}

	s.router.Handle("/metadata", chainHandlers(http.HandlerFunc(m.Get),
		[]HandlerFunc{
			isAuthed,
		}...,
	)).Methods("GET")

	s.router.Handle("/terminate", chainHandlers(http.HandlerFunc(c.Delete),
		[]HandlerFunc{
			isAuthed,
			jwtSignedForTerminate,
		}...,
	)).Methods("DELETE")
}

// HandlerFunc defines the function signature for a middleware function
type HandlerFunc func(http.Handler) http.Handler

// This allows us to chain together middleware to use per route
func chainHandlers(h http.Handler, middleware ...HandlerFunc) http.Handler {
	for _, m := range middleware {
		h = m(h)

		if h == nil {
			break
		}
	}

	return h
}
