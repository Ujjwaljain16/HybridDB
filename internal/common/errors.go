package common

import "errors"

var (
	ErrNotImplemented     = errors.New("not implemented")
	ErrPageNotFound       = errors.New("page not found")
	ErrPageCorrupted      = errors.New("page corrupted")
	ErrInvalidLSN         = errors.New("invalid log sequence number")
	ErrInvariantViolation = errors.New("invariant violation")
	ErrBufferPoolFull     = errors.New("buffer pool full")
	ErrInvalidTuple       = errors.New("invalid tuple")
)
