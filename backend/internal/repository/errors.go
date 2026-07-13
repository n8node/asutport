package repository

import "errors"

var ErrNotFound = errors.New("not found")

var ErrConflict = errors.New("conflict")

var ErrForbidden = errors.New("forbidden")
