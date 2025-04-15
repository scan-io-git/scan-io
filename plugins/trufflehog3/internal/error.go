package internal

import (
	"errors"
	"fmt"
)

var ErrDecodeFailure = errors.New("failed to decode .trufflehog3.yml configuration file")

// WrapDecodeFailure adds context to ErrDecodeFailure
func WrapDecodeFailure(err error) error {
	return fmt.Errorf("%w: %v", ErrDecodeFailure, err)
}
