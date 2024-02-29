package tenet

import "errors"

var (
	ErrNaddr           = errors.New("incomplete naddr")
	ErrEmptyIdentifier = errors.New("no identifier specified")
	ErrEmptyPubKey     = errors.New("no pubkey specified")
)
