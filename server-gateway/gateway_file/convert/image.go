package convert

import (
    "io"
    "fmt"
    "math"
    "os/exec"
)

type Image struct {
}

type ImageMeta struct {
    Width          int64 `json:"width"`
    Height         int64 `json:"height"`
    OriginalWidth  int64 `json:"original_width"`
    OriginalHeight int64 `json:"original_height"`
    Orientation    int   `json:"orientation"`
}

func (c Image) Meta(r io.Reader) (*ImageMeta, error) {
    output := &ImageMeta{}
    iFilename := "-" // STDIN

    // --Init Commands

    // Main Command: identify -format %wx%h -
    cmdMain := exec.Command(_Commands.Identify, "-format", "%wx%h:%[EXIF:Orientation]", iFilename)
    cmdMain.Stdin = r // Command Stdin: Input io.Reader

    // --Start Commands

    var a int
    if b, err := cmdMain.Output(); err != nil {
        _Log.Warn(err.Error())
        return nil, err

    } else if _, err := fmt.Sscanf(string(b), "%dx%d", &output.Width, &output.Height); err != nil {
        _Log.Warn(err.Error())
        return nil, err

    } else if _, err := fmt.Sscanf(string(b), "%vx%v:%d", &a, &a, &output.Orientation); err != nil {
        _Log.Warn(err.Error())
    }

    switch output.Orientation {
    case 5, 6, 7, 8:
        output.OriginalWidth = output.Height
        output.OriginalHeight = output.Width

    default:
        output.OriginalWidth = output.Width
        output.OriginalHeight = output.Height
    }

    return output, nil
}

func (c Image) ToJpeg(r io.Reader, maxWidth, maxHeight uint) (io.Reader, error) {
    iFilename := "-" // STDIN
    oFilename := "-" // STDOUT

    // --Init Commands

    // Main Command:

    args := []string{
        "-auto-orient",
        "-depth", "8",
    }
    if 0 == maxWidth&maxHeight {
        maxDim := uint(math.Max(float64(maxWidth), float64(maxHeight)))
        args = append(args, "-define", fmt.Sprintf("jpeg:size=%dx%d", maxDim*2, maxDim*2))
    } else {
        args = append(args, "-define", fmt.Sprintf("jpeg:size=%dx%d", maxWidth*2, maxHeight*2))
    }

    if 0 == maxWidth|maxHeight {
    } else if 0 == maxWidth {
        args = append(args, "-resize", fmt.Sprintf("x%d>", maxHeight))
    } else if 0 == maxHeight {
        args = append(args, "-resize", fmt.Sprintf("%d>", maxWidth))
    } else {
        args = append(args, "-resize", fmt.Sprintf("%dx%d>", maxWidth, maxHeight))
    }

    args = append(args, iFilename, fmt.Sprintf("jpg:%s", oFilename))

    cmdMain := exec.Command(_Commands.Convert, args...)
    cmdMain.Stdin = r // Command Stdin: Input io.Reader

    var or io.Reader
    if pout, err := cmdMain.StdoutPipe(); err != nil {
        _Log.Warn(err.Error())

        return nil, err
    } else {
        or = pout
    }

    if err := cmdMain.Start(); err != nil {
        _Log.Warn(err.Error())
        return nil, err
    }

    return or, nil
}
