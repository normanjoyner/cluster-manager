package server

import (
	"fmt"
	"net/http"

	"github.com/containership/cluster-manager/pkg/request"
	"github.com/containership/cluster-manager/pkg/server/handlers"

	jwt "github.com/dgrijalva/jwt-go"
)

// ContainershipCustomClaim allows us to reference custom properties in the claim
// of a JWT
type ContainershipCustomClaim struct {
	Metadata struct {
		Roles []string `json:"https://containership.io/roles"`
	} `json:"metadata"`
	jwt.StandardClaims
}

func jwtSignedForTerminate(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwtToken, err := getJWT(r.Header.Get("Authorization"))
		if err != nil {
			handlers.RespondWithError(w, http.StatusUnauthorized, "No JWT auth token")
			return
		}

		token, err := jwt.ParseWithClaims(jwtToken, &ContainershipCustomClaim{}, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("JWT Signing method is not HMAC")
			}

			return []byte(""), nil
		})

		// If there is an error parsing the token we should respond with an error
		// unless the error is that the signature is invalid since we are
		// making a request to cloud.api after this it will
		// verify that the token is signed and valid
		if err != nil && err.Error() != jwt.ErrSignatureInvalid.Error() {
			handlers.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}

		if claims, ok := token.Claims.(*ContainershipCustomClaim); ok && !contains(claims.Metadata.Roles, TerminateRole) {
			handlers.RespondWithError(w, http.StatusUnauthorized, "Token has incorrect containership role")
			return
		}

		h.ServeHTTP(w, r)
	})
}

func contains(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}

	return false
}

func isAuthed(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := "/authenticate"
		req, err := request.New(request.CloudServiceAuth, endpoint, "GET", nil)
		if err != nil {
			handlers.RespondWithError(w, http.StatusInternalServerError, err.Error())
		}

		req.AddHeader("Authorization", r.Header.Get("Authorization"))
		_, err = req.Do()
		if err != nil {
			handlers.RespondWithError(w, http.StatusUnauthorized, "Error validating token")
			return
		}

		// Call the next handler, which can be another middleware in the chain, or the final handler.
		h.ServeHTTP(w, r)
	})
}
