package jsonstorage

import "errors"

var (
	ErrNotExist = errors.New("entry does not exist")
	ErrInternal = errors.New("internal error")
)
