package uuid

import (
	uuid "github.com/google/uuid"
	"crypto/sha256"
	"math/big"
)

func V5(s string) string {
	return uuid.NewSHA1(uuid.Nil, []byte(s)).String()
}

const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func Base62(federationContextId, appId, appInstanceId string) string {
	// Hash deterministico
	hash := sha256.Sum256([]byte(federationContextId + appId + appInstanceId))

	// Prendi i byte dell'hash e converti in Base62
	n := new(big.Int).SetBytes(hash[:])

	base := big.NewInt(62)
	var out []byte

	for n.Sign() > 0 {
		mod := new(big.Int)
		n.DivMod(n, base, mod)
		out = append(out, alphabet[mod.Int64()])
	}

	// reverse
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return string(out)
}
