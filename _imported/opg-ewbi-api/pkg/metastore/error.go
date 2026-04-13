package metastore

import (
	"errors"
	"fmt"

	log "github.com/sirupsen/logrus"
	k8scli "sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrAlreadyExists = errors.New("already exists")
var ErrBadRequest = errors.New("bad request")
var ErrInternal = errors.New("internal error")
var ErrNotFound = errors.New("not found")
var ErrUnauthorized = errors.New("unauthorized")

func IsAlreadyExistsError(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

func IsBadRequestError(err error) bool {
	return errors.Is(err, ErrBadRequest)
}

func IsInternalError(err error) bool {
	return errors.Is(err, ErrInternal)
}

func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

func missMatchErr(objectType, objectId, federationContextID string, expected k8scli.Object, got k8scli.Object) error {
	errMsg := fmt.Errorf("failed to get %s with identifier '%s' in federation context '%s'", objectType, objectId, federationContextID)
	log.Errorf("%s: missmatch types, expected %T, got %T", errMsg, expected, got)
	return fmt.Errorf("%s: internal error", errMsg)
}
