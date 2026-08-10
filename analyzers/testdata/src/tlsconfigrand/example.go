package tlsconfigrand

import (
	"crypto/rand"
	"crypto/tls"
	"io"
)

// Flagged: the Rand field is deprecated as of Go 1.27.
func configWithRand(src io.Reader) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS13,
		Rand:       src, // want `tls\.Config\.Rand is deprecated as of Go 1\.27`
	}
}

// Flagged: assigning the field after construction is the same write.
func setRand(cfg *tls.Config, src io.Reader) {
	cfg.Rand = src // want `tls\.Config\.Rand is deprecated as of Go 1\.27`
}

// Flagged: a value receiver reaches the same field.
func setRandOnValue(src io.Reader) tls.Config {
	var cfg tls.Config
	cfg.Rand = src // want `tls\.Config\.Rand is deprecated as of Go 1\.27`

	return cfg
}

// Not flagged: leaving Rand unset draws from crypto/rand.
func configWithoutRand() *tls.Config {
	return &tls.Config{MinVersion: tls.VersionTLS13}
}

// Not flagged: a Rand field on some other type.
type generator struct {
	Rand io.Reader
}

func setGeneratorRand(gen *generator) {
	gen.Rand = rand.Reader
}
