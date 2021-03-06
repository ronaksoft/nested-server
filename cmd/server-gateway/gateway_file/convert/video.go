package convert

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"

	"git.ronaksoft.com/nested/server/pkg/protocol"
)

type Video struct {
}

type VideoMeta struct {
	Width      int64         `json:"width"`
	Height     int64         `json:"height"`
	Duration   time.Duration `json:"duration"`
	VideoCodec string        `json:"video_codec"`
	AudioCodec string        `json:"audio_codec"`
}

func (c Video) Meta(r io.Reader) (*VideoMeta, error) {
	type stream struct {
		Width     int    `json:"width"`
		Height    int    `json:"height"`
		Duration  string `json:"duration"`
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
	}

	type streams struct {
		Streams []stream `json:"streams"`
	}

	output := &VideoMeta{}
	iFilename := "pipe:" // STDIN

	// --Init Commands

	// Main Command: ffprobe -v error -of flat=s=_ -show_entries stream=height,width
	cmdMain := exec.Command(_Commands.Ffprobe, "-v", "error", "-of", "json", "-show_streams", iFilename)
	cmdMain.Stdin = r // Command Stdin: Input io.Reader

	// --Read command output

	var decoder *json.Decoder
	if pout, err := cmdMain.StdoutPipe(); err != nil {
		_Log.Warn(err.Error())
		return nil, err

	} else {
		decoder = json.NewDecoder(pout)
		decoder.UseNumber()
	}

	if err := cmdMain.Start(); err != nil {
		_Log.Warn(err.Error())
		return nil, err
	}

	result := streams{}
	if err := decoder.Decode(&result); err != nil {
		_Log.Warn(err.Error())
		return nil, err

	}

	// --Create output

	for _, s := range result.Streams {
		switch s.CodecType {
		case "video":
			output.Width = int64(s.Width)
			output.Height = int64(s.Height)
			output.VideoCodec = s.CodecName

			if d, err := time.ParseDuration(fmt.Sprintf("%ss", s.Duration)); err == nil {
				output.Duration = d
			}

		case "audio":
			output.AudioCodec = s.CodecName
		}
	}
	return output, nil
}

func (c Video) ToMp4(r io.Reader, vQuality, maxWidth, maxHeight, aBitRate uint) (io.Reader, error) {
	iFilename := "pipe:" // STDIN
	oFilename := "pipe:" // STDIN

	if f, err := ioutil.TempFile(os.TempDir(), "nst_convert_video_"); err != nil {
		_Log.Warn(err.Error())
		return nil, protocol.NewUnknownError(err)

	} else if s, err := f.Stat(); err != nil {
		_Log.Warn(err.Error())
		return nil, protocol.NewUnknownError(err)

	} else if n, err := io.Copy(f, r); err != nil {
		_Log.Warn(err.Error())
		return nil, protocol.NewUnknownError(err)

	} else if 0 == n {
		_Log.Warn("Video::ToMp4 Nothing was written into temp file")
		return nil, protocol.NewUnknownError(nil)

	} else {
		f.Close()
		iFilename = fmt.Sprintf("%s/%s", os.TempDir(), s.Name())
	}

	if f, err := ioutil.TempFile(os.TempDir(), "nst_convert_video_out_"); err != nil {
		_Log.Warn(err.Error())
		return nil, protocol.NewUnknownError(err)

	} else if s, err := f.Stat(); err != nil {
		_Log.Warn(err.Error())
		return nil, protocol.NewUnknownError(err)

	} else {
		f.Close()
		oFilename = fmt.Sprintf("%s/%s", os.TempDir(), s.Name())
	}

	// --Init Commands

	// Main Command: ffmpeg -i INPUT -codec:v libx264 -preset medium -profile:v main -movflags +faststart -codec:a aac -strict -2 -ac 2 -ar 44100 -f mp4 OUTPUT -hide_banner
	args := []string{
		"-i", iFilename,
		"-codec:v", "libx264",
		"-crf", strconv.FormatUint(uint64(vQuality), 10), // [0-51]
		"-preset", "ultrafast",
		"-profile:v", "main",
		"-movflags", "+faststart",
		"-codec:a", "aac", // TODO: Use libfdk_aac instead: https://trac.ffmpeg.org/wiki/Encode/AAC
		"-b:a", fmt.Sprintf("%dk", aBitRate),
		"-ac", "2",
		"-ar", "44100",
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

	if _, err := cmdMain.CombinedOutput(); err != nil {
		_Log.Warn(err.Error())
		return nil, err
	}

	if err := os.Remove(iFilename); err != nil {
		_Log.Warn(err.Error())
	}

	if f, err := os.Open(oFilename); err != nil {
		_Log.Warn(err.Error())
		return nil, err

	} else {
		return f, nil
	}
}
