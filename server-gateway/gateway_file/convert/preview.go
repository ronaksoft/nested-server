package convert

import (
    "io"
    "os"
    "fmt"
    "math"
    "strings"
    "os/exec"
    "io/ioutil"

    "git.ronaksoftware.com/common/server-protocol"
)

type Preview struct {
}

type PreviewMeta struct {
    Width     int64
    Height    int64
    Extension string
}

func (c Preview) Thumbnail(r io.Reader, mimeType string, maxWidth, maxHeight uint) (io.Reader, *PreviewMeta, error) {
    maxDim := uint(math.Max(float64(maxWidth), float64(maxHeight)))
    iFilename := "-" // STDIN
    filename := "-"  // STDOUT
    ext := ".jpg"

    // Make Command Ready
    var cmd *exec.Cmd
    mimeTypes := strings.SplitN(mimeType, "/", 2)
    switch mimeTypes[0] {
    case "image":
        args := []string{
            "-auto-orient",
            "-background", "white",
            "-alpha", "remove",
            "-depth", "8",
        }
        if 0 == maxWidth&maxHeight {
            args = append(args, "-define", fmt.Sprintf("jpeg:size=%dx%d", maxDim*2, maxDim*2))
            args = append(args, "-thumbnail", fmt.Sprintf("%dx%d>", maxDim, maxDim))
        } else {
            args = append(args, "-define", fmt.Sprintf("jpeg:size=%dx%d", maxWidth*2, maxHeight*2))
            args = append(args, "-thumbnail", fmt.Sprintf("%dx%d>", maxWidth, maxHeight)) // TODO: Not sure if it limits both sides when they differ
        }

        switch mimeTypes[1] {
        case "gif":
            args = append(args, fmt.Sprintf("%s[0]", iFilename), fmt.Sprintf("jpg:%s", filename))

        default:
            args = append(args, iFilename, fmt.Sprintf("jpg:%s", filename))
        }

        cmd = exec.Command(_Commands.Convert, args...)
        cmd.Stdin = r // Command Stdin: Input io.Reader
        //cmd.Stderr = os.Stderr

    case "video":
        if f, err := ioutil.TempFile(os.TempDir(), "nst_convert_preview_"); err != nil {
            _Log.Warn(err.Error())
            return nil, nil, protocol.NewUnknownError(err)

        } else if s, err := f.Stat(); err != nil {
            _Log.Warn(err.Error())
            return nil, nil, protocol.NewUnknownError(err)

        } else if n, err := io.Copy(f, r); err != nil {
            _Log.Warn(err.Error())
            return nil, nil, protocol.NewUnknownError(err)

        } else if 0 == n {
            _Log.Warn(err.Error())
            return nil, nil, protocol.NewUnknownError(nil)

        } else {
            f.Close()
            iFilename = fmt.Sprintf("%s/%s", os.TempDir(), s.Name())
        }

        //inputFilename = "pipe:" // STDIN
        args := []string{
            "-i", iFilename,
            "-vframes", "1",
            "-pix_fmt", "pal8",
            "-f", "singlejpeg",
        }

        if 0 == maxWidth|maxHeight {
        } else if 0 == maxWidth {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale=-1:%d", maxHeight))
        } else if 0 == maxHeight {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale=%d:-1", maxWidth))
        } else {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale='if(gt(a,4/3),%d,-1)':'if(gt(a,4/3),-1,%d)'", maxWidth, maxHeight))
        }

        filename = "pipe:"
        args = append(args, filename)
        cmd = exec.Command(_Commands.Ffmpeg, args...)

    case "audio": // Album Art
        //a := float64(maxWidth) / 600.0
        //w := maxWidth
        //h := int(240 * a)
        iFilename = "pipe:" // STDIN
        args := []string{
            "-i", iFilename,
            //"-filter_complex", fmt.Sprintf("showwavespic=colors=#14d769:s=%dx%d", w, h),
            "-f", "singlejpeg",
        }

        if 0 == maxWidth|maxHeight {
        } else if 0 == maxWidth {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale=-1:%d", maxHeight))
        } else if 0 == maxHeight {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale=%d:-1", maxWidth))
        } else {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale='if(gt(a,4/3),%d,-1)':'if(gt(a,4/3),-1,%d)'", maxWidth, maxHeight))
        }

        filename = "pipe:"
        args = append(args, filename)
        cmd = exec.Command(_Commands.Ffmpeg, args...)
        cmd.Stdin = r // Command Stdin: Input io.Reader
        //cmd.Stderr = os.Stderr

    case "application":
        switch mimeTypes[1] {
        case "pdfx": // FIXME: Query it with poppler
            args := []string{
                "-background", "white",
                "-alpha", "remove",
                "-density", "400",
                "-depth", "8",
            }
            if 0 == maxWidth&maxHeight {
                args = append(args, "-thumbnail", fmt.Sprintf("%dx%d>", maxDim, maxDim))
            } else {
                args = append(args, "-thumbnail", fmt.Sprintf("%dx%d>", maxWidth, maxHeight)) // TODO: Not sure if it limits both sides when they differ
            }

            args = append(args, fmt.Sprintf("%s[0]", iFilename), fmt.Sprintf("jpg:%s", filename))
            cmd = exec.Command(_Commands.Convert, args...)

        default:
            return nil, nil, protocol.NewInvalidError([]string{"mime_type"}, nil)
        }

    default:
        return nil, nil, protocol.NewInvalidError([]string{"mime_type"}, nil)
    }

    // Command Stdout: Output io.Reader
    var output io.ReadCloser
    if pOut, err := cmd.StdoutPipe(); err != nil {
        _Log.Warn(err.Error())
        return nil, nil, err

    } else {
        output = pOut
    }

    // Start Command
    if err := cmd.Start(); err != nil {
        _Log.Warn(err.Error())
        return nil, nil, err
    }

    // FIXME: Return result width & height

    return output, &PreviewMeta{Extension: ext}, nil
}

// Resize the input picture
func (c Preview) Resized(r io.Reader, mimeType string, maxWidth, maxHeight uint) (io.Reader, *PreviewMeta, error) {
    maxDim := uint(math.Max(float64(maxWidth), float64(maxHeight)))
    iFilename := "-" // STDIN
    filename := "-"  // STDOUT
    ext := ".jpg"

    var cmd *exec.Cmd
    mimeTypes := strings.SplitN(mimeType, "/", 2)
    switch mimeTypes[0] {
    case "image":
        args := []string{
            "-auto-orient",
            "-depth", "8",
        }
        if 0 == maxWidth&maxHeight {
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

        switch mimeTypes[1] {
        case "png":
            ext = ".png"

            args = append(args, iFilename, fmt.Sprintf("png:%s", filename))

        case "gif":
            ext = ".png"

            args = append(args, fmt.Sprintf("%s[0]", iFilename), fmt.Sprintf("png:%s", filename))

        default:
            args = append(args, iFilename, fmt.Sprintf("jpg:%s", filename))
        }

        cmd = exec.Command(_Commands.Convert, args...)
        cmd.Stdin = r // Command Stdin: Input io.Reader
        //cmd.Stderr = os.Stderr

    case "video":
        if f, err := ioutil.TempFile(os.TempDir(), "nst_convert_preview_"); err != nil {
            _Log.Warn(err.Error())
            return nil, nil, protocol.NewUnknownError(err)

        } else if s, err := f.Stat(); err != nil {
            _Log.Warn(err.Error())
            return nil, nil, protocol.NewUnknownError(err)

        } else if n, err := io.Copy(f, r); err != nil {
            _Log.Warn(err.Error())
            return nil, nil, protocol.NewUnknownError(err)

        } else if 0 == n {
            _Log.Warn( "Nothing was written into temp file")
            return nil, nil, protocol.NewUnknownError(nil)

        } else {
            f.Close()
            iFilename = fmt.Sprintf("%s/%s", os.TempDir(), s.Name())
        }

        args := []string{
            "-i", iFilename,
            "-vframes", "1",
            "-pix_fmt", "pal8",
            "-f", "singlejpeg",
        }

        if 0 == maxWidth|maxHeight {
        } else if 0 == maxWidth {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale=-1:%d", maxHeight))
        } else if 0 == maxHeight {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale=%d:-1", maxWidth))
        } else {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale='if(gt(a,4/3),%d,-1)':'if(gt(a,4/3),-1,%d)'", maxWidth, maxHeight))
        }

        filename = "pipe:"
        args = append(args, filename)
        cmd = exec.Command(_Commands.Ffmpeg, args...)

    case "audio": // Album Art
        //a := float64(maxWidth) / 600.0
        //w := maxWidth
        //h := int(240 * a)
        iFilename = "pipe:" // STDIN
        args := []string{
            "-i", iFilename,
            //"-filter_complex", fmt.Sprintf("showwavespic=colors=#14d769:s=%dx%d", w, h), // Waveform
            "-f", "singlejpeg",
        }

        if 0 == maxWidth|maxHeight {
        } else if 0 == maxWidth {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale=-1:%d", maxHeight))
        } else if 0 == maxHeight {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale=%d:-1", maxWidth))
        } else {
            args = append(args, "-filter:v", fmt.Sprintf("yadif,scale='if(gt(a,4/3),%d,-1)':'if(gt(a,4/3),-1,%d)'", maxWidth, maxHeight))
        }

        filename = "pipe:"
        args = append(args, filename)
        cmd = exec.Command(_Commands.Ffmpeg, args...)
        cmd.Stdin = r // Command Stdin: Input io.Reader
        //cmd.Stderr = os.Stderr

    case "application":
        switch mimeTypes[1] {
        case "pdfx": // FIXME: Query it with poppler
            args := []string{
                "-background", "white",
                "-alpha", "remove",
                "-density", "400",
                "-depth", "8",
            }

            if 0 == maxWidth|maxHeight {
            } else if 0 == maxWidth {
                args = append(args, "-resize", fmt.Sprintf("x%d>", maxHeight))
            } else if 0 == maxHeight {
                args = append(args, "-resize", fmt.Sprintf("%d>", maxWidth))
            } else {
                args = append(args, "-resize", fmt.Sprintf("%dx%d>", maxWidth, maxHeight))
            }

            args = append(args, fmt.Sprintf("%s[0]", iFilename), fmt.Sprintf("jpg:%s", filename))
            cmd = exec.Command(_Commands.Convert, args...)

        default:
            return nil, nil, protocol.NewInvalidError([]string{"mime_type"}, nil)
        }

    default:
        return nil, nil, protocol.NewInvalidError([]string{"mime_type"}, nil)
    }

    // Command Stdout: Output io.Reader
    var output io.ReadCloser
    if pOut, err := cmd.StdoutPipe(); err != nil {
        _Log.Warn(err.Error())
        return nil, nil, err

    } else {
        output = pOut
    }

    // Start Command
    if err := cmd.Start(); err != nil {
        _Log.Warn(err.Error())
        return nil, nil, err
    }

    // FIXME: Return result width & height

    return output, &PreviewMeta{Extension: ext}, nil
}
