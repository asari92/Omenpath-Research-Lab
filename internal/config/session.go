package config

import (
	"fmt"
	"strconv"
)

// CookieSecure parses OMENPATH_COOKIE_SECURE. Production must explicitly opt
// into secure cookies; an invalid value never silently disables protection.
func CookieSecure(value string, production bool) (bool, error) {
	secure := false
	if value != "" {
		var err error
		secure, err = strconv.ParseBool(value)
		if err != nil {
			return false, fmt.Errorf("OMENPATH_COOKIE_SECURE must be a boolean")
		}
	}
	if production && !secure {
		return false, fmt.Errorf("production requires OMENPATH_COOKIE_SECURE=true")
	}
	return secure, nil
}
