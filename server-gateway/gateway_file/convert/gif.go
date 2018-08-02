package convert

import (
    "io"
    "os"
    "fmt"
    "strconv"
    "os/exec"
    "io/ioutil"

    "git.ronaksoftware.com/common/server-protocol"
)

type Gif struct {
}

type GifMeta struct {
    Width  int64 `json:"width"`
    Height int64 `json:"height"`
    Frames uint  `json:"frames"`
}

func (c Gif) Meta(r io.Reader) (*GifMeta, error) {
    _funcName := "Gif::Meta"

    output := &GifMeta{}
    iFilename := "-[-1]" // STDIN

    // --Init Commands

    // Main Command: identify -format %wx%h -
    cmdMain := exec.Command(_Commands.Identify, "-format", "%[scene]x%wx%h", iFilename)
    cmdMain.Stdin = r // Command Stdin: Input io.Reader

    // --Start Commands

    if b, err := cmdMain.Output(); err != nil {
        _Log.Error(_funcName, err.Error())

        return nil, err
    } else if _, err := fmt.Sscanf(string(b), "%dx%dx%d", &output.Frames, &output.Width, &output.Height); err != nil {
        _Log.Error(_funcName, err.Error(), string(b))

        return nil, err
    }
    output.Frames = output.Frames + 1
    return output, nil
}

func (c Gif) ToMp4(r io.Reader, vQuality, maxWidth, maxHeight uint) (io.Reader, error) {
    _funcName := "Gif::ToMp4"

    iFilename := "pipe:" // STDIN
    oFilename := "pipe:" // STDIN

    if f, err := ioutil.TempFile(os.TempDir(), "nst_convert_gif_"); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil, protocol.NewUnknownError(err)

    } else if s, err := f.Stat(); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil, protocol.NewUnknownError(err)

    } else if n, err := io.Copy(f, r); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil, protocol.NewUnknownError(err)

    } else if 0 == n {
        _Log.Error(_funcName, "Gif::ToMp4 Nothing was written into temp file")
        return nil, protocol.NewUnknownError(nil)

    } else {
        f.Close()
        iFilename = fmt.Sprintf("%s/%s", os.TempDir(), s.Name())
    }

    if f, err := ioutil.TempFile(os.TempDir(), "nst_convert_gif_out_"); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil, protocol.NewUnknownError(err)

    } else if s, err := f.Stat(); err != nil {
        _Log.Error(_funcName, err.Error())
        return nil, protocol.NewUnknownError(err)

    } else {
        f.Close()
        oFilename = fmt.Sprintf("%s/%s", os.TempDir(), s.Name())
    }

    // --Init Commands

    // Main Command: ffmpeg -i INPUT -codec:v libx264 -preset medium -movflags +faststart -f mp4 OUTPUT -hide_banner
    args := []string{
        "-i", iFilename,
        "-codec:v", "libx264",
        "-crf", strconv.FormatUint(uint64(vQuality), 10), // [0-51]
        "-preset", "medium",
        "-movflags", "+faststart",
        "-an",
        "-f", "mp4",
        "-hide_banner",
        "-y",
    }

    if 0 == maxWidth|maxHeight {
    } else if 0 == maxWidth {
        args = append(args, "-filter:v", fmt.Sprintf("yadif,scale=-1:'if(gt(ih,%d),%d,-1)'", maxHeight, maxHeight))
    } else if 0 == maxHeight {
        args = append(args, "-filter:v", fmt.Sprintf("yadif,scale='if(gt(iw,%d),%d,-1)':-1", maxWidth, maxWidth))
    } else {
        args = append(args, "-filter:v", fmt.Sprintf("yadif,scale='if(gt(a,4/3),if(gt(iw,%d),%d,iw),-1)':'if(gt(a,4/3),-1,if(gt(ih,%d),%d,-1))'", maxWidth, maxWidth, maxHeight, maxHeight))
    }

    args = append(args, oFilename)
    cmdMain := exec.Command(_Commands.Ffmpeg, args...)
    cmdMain.Stdin = r // Command Stdin: Input io.Reader

    if b, err := cmdMain.CombinedOutput(); err != nil {
        _Log.Error(_funcName, err.Error(), cmdMain.Path, b)
        return nil, err

    }

    if err := os.Remove(iFilename); err != nil {
        _Log.Error(_funcName,  err.Error())
    }

    if f, err := os.Open(oFilename); err != nil {
        _Log.Error(_funcName,  err.Error())
        return nil, err

    } else {
        return f, nil
    }
}
