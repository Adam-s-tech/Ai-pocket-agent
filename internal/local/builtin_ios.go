//go:build ios && llama && cgo

package local

/*
#cgo CFLAGS: -I${SRCDIR}/../../third_party/llama.cpp/include -I${SRCDIR}/../../third_party/llama.cpp/ggml/include -O2
#cgo LDFLAGS: -L${SRCDIR}/../../third_party/lib/ios_arm64 -lllama -lggml -lggml-cpu -lggml-metal -lggml-blas -lggml-base -lc++ -framework Foundation -framework Metal -framework MetalKit -framework Accelerate
*/
import "C"

// builtinEngine is set when the engine comes with the app rather than
// being downloaded: here it's linked in (iOS, built with -tags llama).
const builtinEngine = true

// packagedLibrary is "": the linked-in engine needs no library.
func packagedLibrary() string { return "" }
