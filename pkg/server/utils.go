package server

import (
	"fmt"
	"strings"
)

func getJWT(a string) (string, error) {
	if a == "" {
		return "", fmt.Errorf("Authorization header is empty")
	}

	jwtToken := strings.Split(a, " ")
	if len(jwtToken) != 2 {
		return "", fmt.Errorf("JWT token is ill formated")
	}

	return jwtToken[1], nil
}
