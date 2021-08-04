package convert

import (
	"encoding/json"
	"fmt"
	"git.ronaksoft.com/nested/server/pkg/log"
	"git.ronaksoft.com/nested/server/pkg/protocol"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type Voice struct{}

type VoiceMeta struct {
	Duration   time.Duration `json:"duration" bson:"duration"`
	AudioCodec string        `json:"audio_codec" bson:"audio_codec"`
}

func (c Voice) Meta(r io.Reader) (*VoiceMeta, error) {
	type stream struct {
		Duration  string `json:"duration"`
		CodecType string `json:"codec_type"`
		CodecName string `json:"codec_name"`
	}

	type streams struct {
		Streams []stream `json:"streams"`
	}

	output := &VoiceMeta{}
	inputFilename := "pipe:" // STDIN

	// --Init Commands

	// Main Command: ffprobe -v error -of json -show_streams INPUT
	cmdMain := exec.Command(_Commands.Ffprobe, "-v", "error", "-of", "json", "-show_streams", inputFilename)
	cmdMain.Stdin = r // Command Stdin: Input io.Reader

	// --Read command output
	var decoder *json.Decoder
	if pout, err := cmdMain.StdoutPipe(); err != nil {
		log.Warn(err.Error())
		return nil, err

	} else {
		decoder = json.NewDecoder(pout)
		decoder.UseNumber()
	}

	if err := cmdMain.Start(); err != nil {
		log.Warn(err.Error())
		return nil, err
	}

	result := streams{}
	if err := decoder.Decode(&result); err != nil {
		log.Warn(err.Error())
		return nil, err
	}
	// --Create output

	for _, s := range result.Streams {
		switch s.CodecType {
		case "audio":
			output.AudioCodec = s.CodecName
			if d, err := time.ParseDuration(fmt.Sprintf("%ss", s.Duration)); err == nil {
				output.Duration = d
			}
		}
	}

	return output, nil
}

func (c Voice) ToMp3(r io.Reader, aQuality uint) (io.Reader, error) {

	iFilename := "pipe:" // STDIN
	oFilename := "-"     // STDIN

	if f, err := ioutil.TempFile(os.TempDir(), "nst_convert_voice_"); err != nil {
		log.Warn(err.Error())
		return nil, protocol.NewUnknownError(err)

	} else if s, err := f.Stat(); err != nil {
		log.Warn(err.Error())
		return nil, protocol.NewUnknownError(err)

	} else if n, err := io.Copy(f, r); err != nil {
		log.Warn(err.Error())
		return nil, protocol.NewUnknownError(err)

	} else if 0 == n {
		log.Warn("Voice::ToMp3 Nothing was written into temp file")
		return nil, protocol.NewUnknownError(nil)

	} else {
		f.Close()
		iFilename = fmt.Sprintf("%s/%s", os.TempDir(), s.Name())
	}

	// --Init Commands

	// Main Command: ffmpeg -i INPUT -vn -codec:a libmp3lame -ar 44100 -ac 2 -q:a QUALITY -f mp3 OUTPUT
	args := []string{
		"-i", iFilename,
		"-vn",
		"-codec:a", "libmp3lame",
		"-ar", "44100",
		"-ac", "2",
		"-q:a", strconv.FormatUint(uint64(aQuality), 10), // [0-9]
		"-f", "mp3",
		oFilename,
	}

	cmdMain := exec.Command(_Commands.Ffmpeg, args...)
	cmdMain.Stdin = r // Command Stdin: Input io.Reader

	var or io.Reader
	if pout, err := cmdMain.StdoutPipe(); err != nil {
		log.Warn(err.Error())
		return nil, err

	} else {
		or = pout
	}

	if err := cmdMain.Start(); err != nil {
		log.Warn(err.Error())
		return nil, err
	}

	return or, nil

}
