package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cocktailrobots/say"
)

type SayArgs struct {
	filename string
}

func errExit(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	ctx := context.Background()
	sayArgs := validateArgs()
	rd, err := say.ReaderForFile(ctx, sayArgs.filename, 0)
	errExit(err)

	const numChars = 20
	printAmplitude(0, numChars, false)
	err = say.PlayWithCallback(rd, 50*time.Millisecond, func(amplitude float64) error {
		printAmplitude(amplitude, numChars, true)
		return nil
	})

	errExit(err)
}

func validateArgs() SayArgs {
	if len(os.Args) != 2 {
		errExit(errors.New("Usage: say <filename>"))
	}

	return SayArgs{
		filename: os.Args[1],
	}
}

func printAmplitude(amplitude float64, chars int, clear bool) {
	var clrStr string
	if clear {
		for i := 0; i < chars; i++ {
			clrStr += "\b"
		}
	}

	var s string
	chars = chars / 2
	for i := 0; i < chars; i++ {
		if float64(i)/float64(chars) < amplitude {
			s += "#"
		} else {
			s += " "
		}
	}

	reversedS := ""
	for _, c := range s {
		reversedS = string(c) + reversedS
	}

	fmt.Print(clrStr + reversedS + s)
}
