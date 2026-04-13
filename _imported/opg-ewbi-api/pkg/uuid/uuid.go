package uuid

import uuid "github.com/google/uuid"

func V5(s string) string {
	return uuid.NewSHA1(uuid.Nil, []byte(s)).String()
}
