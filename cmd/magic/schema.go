package main

import "magic/internal/core"

// requireLibrarySchema refuses to run a command on a library this magic cannot
// read: a schema-0 library needs `magic migrate` first, and a library written by
// a newer magic must not be written by this one. doctor reports the same
// three-way state as a section instead of failing up front.
func requireLibrarySchema(magicHome string) error {
	_, err := core.CheckLibrarySchema(core.NewPaths(magicHome).MagicYAML)
	return err
}
