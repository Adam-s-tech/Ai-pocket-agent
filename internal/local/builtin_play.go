//go:build android && arm64 && play && cgo

package local

// builtinEngine is set when the engine comes with the app rather than
// being downloaded: Google Play builds (-tags play) package it, as Play
// doesn't let apps download native code.
const builtinEngine = true

// The engine builds packaged in a Play build's lib/arm64-v8a, one per
// CPU variant (scripts/engine-aar.sh).
var packagedLibraries = map[string]string{
	"armv8.2": "libpocketllama_armv82.so",
	"armv8":   "libpocketllama_armv8.so",
}

// packagedLibrary is the packaged engine for this phone's CPU. The bare
// name is found in the app's native library directory.
func packagedLibrary() string {
	return packagedLibraries[cpuVariants()[0]]
}
