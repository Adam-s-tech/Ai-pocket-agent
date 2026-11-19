//go:build !(ios && llama && cgo) && !(android && arm64 && play && cgo)

package local

// builtinEngine is set when the engine comes with the app rather than
// being downloaded: linked in on iOS (-tags llama), packaged in Google
// Play builds (-tags play).
const builtinEngine = false

// packagedLibrary is the engine library that comes with the app, ""
// when it's downloaded.
func packagedLibrary() string { return "" }
