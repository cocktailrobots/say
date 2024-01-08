package wav

import (
	"io"
	"os"
	"sync"

	"github.com/cocktailrobots/say/reader"
	"github.com/ebitengine/oto/v3"
)

const HdrSize = 44

var _ reader.SayReader = (*Reader)(nil)

type Reader struct {
	mu        *sync.Mutex
	data      [][]byte
	available int
	readPos   int

	inStr   io.ReadCloser
	readErr error
}

func NewReader(rc io.ReadCloser) *Reader {
	return &Reader{
		mu:    &sync.Mutex{},
		inStr: rc,
	}
}

func (r *Reader) Start() {
	const bufSize = 128 * 1024
	go func() {
		for {
			buf := make([]byte, bufSize)
			n, err := r.inStr.Read(buf)

			func() {
				r.mu.Lock()
				defer r.mu.Unlock()

				if n > 0 {
					r.data = append(r.data, buf[0:n])
					r.available += n
				}

				if r.readErr == nil {
					r.readErr = err
				}
			}()

			if err != nil {
				break
			}
		}
	}()
}

func (r *Reader) Read(p []byte) (n int, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	desired := len(p)
	read := r.readAt(p, r.readPos)
	r.readPos += read

	if read < desired {
		return read, r.readErr
	} else {
		return read, nil
	}
}

func (r *Reader) readAt(p []byte, off int) (n int) {
	desired := len(p)

	var bufIdx int
	var bufPos int
	for pos := 0; pos < off; {
		if pos+len(r.data[bufIdx]) > off {
			bufPos = off - pos
			pos = off
		} else {
			pos += len(r.data[bufIdx])
			bufIdx += 1
		}
	}

	var read int
	for read < desired && bufIdx < len(r.data) {
		remainingToRead := desired - read
		remainingInBuf := len(r.data[bufIdx]) - bufPos

		if remainingInBuf > 0 {
			if remainingInBuf > remainingToRead {
				copy(p[read:], r.data[bufIdx][bufPos:bufPos+remainingToRead])
				read += remainingToRead
				continue
			} else {
				copy(p[read:], r.data[bufIdx][bufPos:])
				read += remainingInBuf
			}
		}

		bufIdx += 1
		bufPos = 0
	}

	return read
}

func (r *Reader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.readErr == nil {
		r.readErr = os.ErrClosed
	}

	return r.inStr.Close()
}

// GetSampleRate returns the sample rate of the audio
func (r *Reader) GetSampleRate() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.available > HdrSize {
		var data [4]byte
		n := r.readAt(data[:], 24)
		if n < 4 {
			panic("Failed to read sample rate despite having enough data")
		}

		return int(data[0]) + int(data[1])<<8 + int(data[2])<<16 + int(data[3])<<24
	} else {
		return -1
	}
}

// GetNumChans returns the number of channels in the audio
func (r *Reader) GetNumChans() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.available > HdrSize {
		var data [2]byte
		n := r.readAt(data[:], 22)
		if n < 2 {
			panic("Failed to read num channels despite having enough data")
		}

		return int(data[0]) + int(data[1])<<8
	} else {
		return 0
	}
}

// GetFormat returns the format of the audio
func (r *Reader) GetFormat() (oto.Format, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.available > HdrSize {
		var data [4]byte
		n := r.readAt(data[:], 0)
		if n < 4 {
			panic("Failed to read format despite having enough data")
		}

		if data[0] == 'R' && data[1] == 'I' && data[2] == 'F' && data[3] == 'F' {
			n := r.readAt(data[:2], 34)
			if n < 2 {
				panic("Failed to read bits per sample despite having enough data")
			}

			bitsPerSample := int(data[0]) + int(data[1])<<8
			switch bitsPerSample {
			case 16:
				return oto.FormatSignedInt16LE, nil
			//case 8:
			//	return oto.FormatUnsignedInt8, nil
			//case 32:
			//	return oto.FormatFloat32LE, nil
			default:
				return 0, reader.ErrUnsupportedFormat
			}
		} else {
			return 0, reader.ErrUnsupportedFormat
		}
	} else {
		return 0, reader.ErrUnsupportedFormat
	}
}

// AmplitudeAtPos returns the amplitude at the given position
func (r *Reader) AmplitudeAtPos(pos int) (float64, error) {
	const desiredDataPoints = 32
	var dataPoints []int16
	numChans := r.GetNumChans()
	for len(dataPoints) < desiredDataPoints {
		dataPointPos := pos + len(dataPoints)*2*numChans
		if dataPointPos >= r.available {
			break
		}

		var data [2]byte
		n := r.readAt(data[:], dataPointPos)
		if n < 2 {
			panic("Failed to read amplitude despite having enough data")
		}

		a := int16(data[0]) + int16(data[1])<<8
		dataPoints = append(dataPoints, a)
	}

	if len(dataPoints) == 0 {
		return 0, io.EOF
	}

	var maxAmplitude float64
	for _, dp := range dataPoints {
		normalized := float64(dp) / 32768.0
		if normalized < 0 {
			normalized = -normalized
		}

		if normalized > maxAmplitude {
			maxAmplitude = normalized
		}
	}

	return maxAmplitude, nil
}

// BytesAvailable returns the number of bytes available to read
func (r *Reader) BytesAvailable() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.available
}

// GetPos returns the number of bytes that
func (r *Reader) GetPos() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.readPos
}
