package say

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cocktailrobots/say/reader"
	"github.com/cocktailrobots/say/reader/wav"
	"github.com/ebitengine/oto/v3"
)

func ReaderForFile(filename string) (reader.SayReader, error) {
	if filepath.Ext(filename) == ".wav" {
		f, err := openFile(filename)
		if err != nil {
			return nil, err
		}

		wavRd := wav.NewReader(f)
		wavRd.Start()

		for {
			if wavRd.BytesAvailable() < wav.HdrSize {
				time.Sleep(time.Millisecond)
			} else {
				break
			}
		}

		n, err := wavRd.Read(make([]byte, wav.HdrSize))
		if err != nil {
			wavRd.Close()
			return nil, fmt.Errorf("failed to read header: %w", err)
		} else if n != wav.HdrSize {
			wavRd.Close()
			return nil, fmt.Errorf("failed to read header: short read of %n bytes instead of %n", n, wav.HdrSize)
		}

		return wavRd, nil
	} else {
		return nil, fmt.Errorf("unsupported file type '%s'", filepath.Ext(filename))
	}
}

func openFile(filename string) (*os.File, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file '%s': %w", filename, err)
	}

	return f, nil
}

func PlayWithCallback(rdr reader.SayReader, sleepDur time.Duration, cb func(amplitude float64) error) error {
	otoFmt, err := rdr.GetFormat()
	if err != nil {
		return fmt.Errorf("failed to get format: %w", err)
	}

	ctxOpts := oto.NewContextOptions{
		SampleRate:   rdr.GetSampleRate(),
		ChannelCount: rdr.GetNumChans(),
		Format:       otoFmt,
	}

	otoCtx, otoReadyCh, err := oto.NewContext(&ctxOpts)
	if err != nil {
		return fmt.Errorf("failed to create oto context: %w", err)
	}

	<-otoReadyCh

	p := otoCtx.NewPlayer(rdr)
	p.Play()
	defer p.Close()

	for p.IsPlaying() {
		pos := rdr.GetPos()
		remaining := p.BufferedSize()

		amplitudePos := pos - remaining
		amplitude, err := rdr.AmplitudeAtPos(amplitudePos)

		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to get amplitude at pos %d: %w", amplitudePos, err)
		}

		amplitude = reader.NormalizeInRange(amplitude, 0.1, 0.8)
		err = cb(amplitude)
		if err != nil {
			return fmt.Errorf("callback failed: %w", err)
		}

		time.Sleep(sleepDur)
	}

	return nil
}
