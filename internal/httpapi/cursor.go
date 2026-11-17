package httpapi

import (
	"encoding/base64"
	"errors"
	"strings"
)

// The cursor is the (due date, ID) of the last invoice on a page, encoded
// as one opaque string. Clients pass it back unchanged to get the next page.
//
// Base64 is not a secret: anyone can decode it. Encoding only tells clients
// "don't build or edit this yourself", so the format can change later
// without breaking them. The server still validates whatever comes back.

var errBadCursor = errors.New("invalid cursor")

func encodeCursor(dueDate, id string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(dueDate + "|" + id))
}

func decodeCursor(s string) (dueDate, id string, err error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return "", "", errBadCursor
	}
	dueDate, id, ok := strings.Cut(string(b), "|")
	if !ok || dueDate == "" || id == "" {
		return "", "", errBadCursor
	}
	return dueDate, id, nil
}
