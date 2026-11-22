// Package api declares an interface another fixture package satisfies.
package api

// Stopper is declared outside the package that implements it.
type Stopper interface{ Shutdown() error }
