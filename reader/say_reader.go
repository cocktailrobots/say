package reader

import (
	"errors"
	"io"

	"github.com/ebitengine/oto/v3"
)

var ErrUnsupportedFormat = errors.New("unsupported format")

type SayReader interface {
	io.ReadCloser

	// GetSampleRate returns the sample rate of the audio
	GetSampleRate() int
	// GetNumChans returns the number of channels in the audio
	GetNumChans() int
	// AmplitudeAtPos returns the amplitude at the given position
	AmplitudeAtPos(pos int) (float64, error)
	// BytesAvailable returns the number of bytes available to read
	BytesAvailable() int
	// GetFormat returns the format of the audio
	GetFormat() (oto.Format, error)
	// GetPos returns the number of bytes that
	GetPos() int
}

func NormalizeInRange(val, cutoff, theshold float64) float64 {
	if val < cutoff {
		return 0
	} else if val > theshold {
		return 1
	} else {
		return (val - cutoff) / (theshold - cutoff)
	}
}
