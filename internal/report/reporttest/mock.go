package reporttest

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

// block is what every endpoint of Mock hashes: 4 KiB of a fixed pattern.
var block = func() (b [4096]byte) {
	for i := range b {
		b[i] = byte(i * 31)
	}

	return b
}()

// mockWork is how many times each endpoint of Mock feeds block to one SHA-256
// for a request: about 4 ms, 0.2 s and 1 s of one core on the machine the live
// recordings were made on.
var mockWork = []struct {
	path   string
	blocks int
}{
	{"/{$}", 2_000},
	{"/report", 128_000},
	{"/export", 640_000},
}

// Mock returns the service the live Gatling runs load. Every endpoint does real
// work for every request — it feeds a fixed 4 KiB block to one SHA-256 a fixed
// number of times — and answers 200 with the first eight bytes of the sum.
// Nothing in it sleeps, reads a clock or draws a random number, so a request
// takes as long as its work and the requests ahead of it for a core, and every
// response time of a run it serves comes from the machine and the load. Any
// other path is answered 404, and any other method 405 before any work.
func Mock() http.Handler {
	mux := http.NewServeMux()

	for _, endpoint := range mockWork {
		// The method is checked here and not in the pattern: a GET pattern also
		// matches HEAD, which would do the whole work and answer 200.
		mux.HandleFunc(endpoint.path, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)

				return
			}

			sum := sha256.New()
			for range endpoint.blocks {
				sum.Write(block[:])
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"digest":"` + hex.EncodeToString(sum.Sum(nil)[:8]) + `"}`))
		})
	}

	return mux
}
