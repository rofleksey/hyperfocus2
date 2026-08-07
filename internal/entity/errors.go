package entity

import "errors"

// ErrNotFound is the shared sentinel returned by repository lookups when no row
// matches. Declared in the entity layer so both repositories and usecases can
// reference it without depending on a concrete adapter package.
var ErrNotFound = errors.New("entity: not found")
