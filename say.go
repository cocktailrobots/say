package say

import (
	"context"
	"errors"
	"fmt"
	"github.com/cocktailrobots/say/reader/wav"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cocktailrobots/say/reader"
	"github.com/ebitengine/oto/v3"
)

type AudioFileType int

const (
	AudioFileTypeWav AudioFileType = iota
)

func ReaderForFile(ctx context.Context, filename string, preloadDuration time.Duration) (reader.SayReader, error) {
	var f *os.File
	if filepath.Ext(filename) == ".wav" {
		var err error
		f, err = openFile(filename)
		if err != nil {
			return nil, err
		}

		return ReaderForStream(ctx, f, AudioFileTypeWav, preloadDuration)
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

func ReaderForStream(ctx context.Context, rc io.ReadCloser, fileType AudioFileType, preloadDuration time.Duration) (reader.SayReader, error) {
	switch fileType {
	case AudioFileTypeWav:
		rdr := wav.NewReader(rc)
		err := rdr.Start(ctx, preloadDuration)
		if err != nil {
			rdr.Close()
			return nil, fmt.Errorf("failed to start wav reader: %w", err)
		}

		return rdr, nil
	default:
		return nil, fmt.Errorf("unsupported file type '%d'", fileType)
	}
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
