package convert

import (
	"fmt"
	"io"
	"os/exec"
)

type Pdf struct {
}

type PdfMeta struct {
	Width     float32 `json:"width"`
	Height    float32 `json:"height"`
	PageCount int     `json:"page_count"`
}

func (c Pdf) Meta(r io.Reader) (*PdfMeta, error) {
	output := &PdfMeta{}
	inputFilename := "-" // STDIN

	// --Init Commands

	// Main Command: pdfinfo -
	cmdMain := exec.Command(_Commands.PdfInfo, inputFilename)
	cmdMain.Stdin = r // Command Stdin: Input io.Reader

	// Pipe 1: grep "Pages:\|Page size:"
	cmdPipe1 := exec.Command(_Commands.Grep, "Pages:\\|Page size:")
	if pin, err := cmdMain.StdoutPipe(); err != nil {
		return nil, err

	} else {
		cmdPipe1.Stdin = pin
	}

	// Pipe 2: sed "s/.\+://"
	cmdPipe2 := exec.Command(_Commands.Sed, "s/.\\+://")
	if pin, err := cmdPipe1.StdoutPipe(); err != nil {
		return nil, err

	} else {
		cmdPipe2.Stdin = pin
	}

	// Pipe 3: awk '{$1=$1};1'
	cmdPipe3 := exec.Command(_Commands.Awk, "{$1=$1};1")
	if pin, err := cmdPipe2.StdoutPipe(); err != nil {
		return nil, err

	} else {
		cmdPipe3.Stdin = pin
	}

	// Pipe 4: tr "\n" " "
	cmdPipe4 := exec.Command(_Commands.Tr, "\n", " ")
	if pin, err := cmdPipe3.StdoutPipe(); err != nil {
		return nil, err

	} else {
		cmdPipe4.Stdin = pin
	}

	// --Start Commands

	if err := cmdMain.Start(); err != nil {
		return nil, err
	}

	if err := cmdPipe1.Start(); err != nil {
		return nil, err
	}

	if err := cmdPipe2.Start(); err != nil {
		return nil, err
	}

	if err := cmdPipe3.Start(); err != nil {
		return nil, err
	}

	if b, err := cmdPipe4.Output(); err != nil {
		return nil, err

	} else if _, err := fmt.Sscanf(string(b), "%d %f x %f", &output.PageCount, &output.Width, &output.Height); err != nil {
		return nil, err
	}

	return output, nil
}
