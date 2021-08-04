package nested

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/globalsign/mgo/bson"
)

func Timestamp() (ts uint64) {
	ts = uint64(time.Now().UnixNano() / 1000000)
	return
}

func RandomID(n int) string {
	rand.Seed(time.Now().UnixNano())
	ts := strings.ToUpper(strconv.FormatInt(time.Now().UnixNano(), 16))
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const size = 36
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(size)]
	}
	return fmt.Sprintf("%s%s", ts, string(b))
}

func RandomPassword(n int) string {
	rand.Seed(time.Now().UnixNano())
	ts := strings.ToUpper(strconv.FormatInt(time.Now().UnixNano(), 16))
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuvwxyz"
	const size = 62
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(size)]
	}
	return fmt.Sprintf("%s%s", ts, string(b))
}

func RandomDigit(n int) string {
	rand.Seed(time.Now().UnixNano())
	const letters = "0123456789"
	const size = 10
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(size)]
	}
	return fmt.Sprintf("%s", string(b))
}

func Encrypt(keyText, text string) string {
	key := []byte(keyText)
	rand.Seed(time.Now().UnixNano())
	plaintext := []byte(text)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Warn(err.Error())
		return ""
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the cipher-text.
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := rand.Read(iv); err != nil {
		log.Info(err.Error())
		return ""
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	// convert to base64
	return base64.URLEncoding.EncodeToString(ciphertext)
}

func Decrypt(keyText, cryptoText string) string {
	key := []byte(keyText)
	ciphertext, _ := base64.URLEncoding.DecodeString(cryptoText)

	block, err := aes.NewCipher(key)
	if err != nil {
		log.Warn(err.Error())
		return ""
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the cipher-text.
	if len(ciphertext) < aes.BlockSize {
		log.Warn("ciphertext too short")
		return ""
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)

	// XORKeyStream can work in-place if the two arguments are the same.
	stream.XORKeyStream(ciphertext, ciphertext)

	return fmt.Sprintf("%s", ciphertext)
}

func ClampInteger(val, min, max int) int {
	if val > max {
		val = max
	} else if val < min {
		val = min
	}
	return val
}

func SystemInfo() M {
	m := new(runtime.MemStats)
	runtime.ReadMemStats(m)
	return M{
		"memory": M{
			"objects_allocated": humanize.Comma(int64(m.Mallocs)),
			"objects_freed":     humanize.Comma(int64(m.Frees)),
			"objects_live":      humanize.Comma(int64(m.Mallocs - m.Frees)),

			"heap_alloc":    humanize.Bytes(m.HeapAlloc), // Not free memory
			"heap_idle":     humanize.Bytes(m.HeapIdle),
			"heap_released": humanize.Bytes(m.HeapReleased),
			"heap_retained": humanize.Bytes(m.HeapIdle - m.HeapReleased),
			"heap_fragment": humanize.Bytes(m.HeapInuse - m.HeapAlloc),
			"heap_objects":  humanize.Comma(int64(m.HeapObjects)),

			"sys":       humanize.Bytes(m.Sys),
			"sys_stack": humanize.Bytes(m.StackSys),
			"sys_heap":  humanize.Bytes(m.HeapSys),
		},
		"go-routines":     humanize.Comma(int64(runtime.NumGoroutine())),
		"gc_total_pause":  humanize.Comma(int64(m.PauseTotalNs / 1000000)),
		"gc_cpu_fraction": humanize.Commaf(m.GCCPUFraction),
	}
}

func IsValidEmail(email string) bool {
	if email != "" {
		if b, err := regexp.MatchString(global.DEFAULT_REGEX_EMAIL, email); err != nil || !b {
			return false
		}
		return true
	}
	return false
}

// UseDownloadToken validates token and returns TRUE and the universalID of the file, otherwise
// returns FALSE
func UseDownloadToken(token string) (bool, UniversalID) {
	// Remove the expire timestamp from the token
	token = string(token[:strings.LastIndex(token, "-")])

	ct := Decrypt(TokenSeedSalt, token)
	p := strings.Split(ct, "/")

	if len(p) > 3 {
		if et, err := strconv.Atoi(p[3]); err != nil {
			log.Warn(err.Error())

			return false, ""
		} else if Timestamp() > uint64(et) {
			return false, UniversalID(p[1])
		}

		return true, UniversalID(p[1])
	}

	return false, ""
}

// UseUploadToken validates upload token and returns the maximum upload size
func UseUploadToken(token string, sk bson.ObjectId) (bool, string) {
	token = string(token[:strings.LastIndex(token, "-")])
	ct := Decrypt(TokenSeedSalt, token)
	p := strings.Split(ct, "/")

	if len(p) > 2 {
		if et, err := strconv.Atoi(p[2]); err != nil {
			log.Warn(err.Error())

			return false, ""
		} else if Timestamp() > uint64(et) {
			return false, ""
		}

		if sk.String() == p[0] {
			return true, p[1]
		}
	}
	return false, ""
}

// GenerateUploadToken generates a new upload token for the session identified by sk
// 1. Session DHKey
// 2. Maximum Upload Size
// 3. Expiry time
func GenerateUploadToken(sk bson.ObjectId) (string, error) {
	et := Timestamp() + TokenLifetime
	tv := fmt.Sprintf("%s/%s/%s", sk.String(), global.DEFAULT_MAX_UPLOAD_SIZE, strconv.Itoa(int(et)))

	token := fmt.Sprintf("%s-%d", Encrypt(TokenSeedSalt, tv), et)
	return token, nil
}

// GenerateDownloadToken generates new token to download a file identified by Universal ID
// 1. Account ID
// 2. Universal ID
// 3. Session DHKey
// 4. Expiry Time
func GenerateDownloadToken(uniID UniversalID, sk bson.ObjectId, accountID string) (string, error) {
	et := Timestamp() + TokenLifetime
	tv := fmt.Sprintf("%s/%s/%s/%s", accountID, string(uniID), sk.String(), strconv.Itoa(int(et)))

	token := fmt.Sprintf("%s-%d", Encrypt(TokenSeedSalt, tv), et)
	return token, nil
}

func GenerateUniversalID(filename string, ftype string) UniversalID {
	if 0 == len(ftype) {
		ftype = GetTypeByFilename(filename)
	}

	return UniversalID(strings.ToUpper(fmt.Sprintf("%s%s%s", ftype, bson.NewObjectId().Hex(), bson.NewObjectId().Hex())))
}

func GetMimeTypeByFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	var mimeType string

	switch ext {
	case ".txt":
		mimeType = "text/plain"
	case ".md":
		mimeType = "text/markdown"

	case ".bmp":
		mimeType = "image/bmp"
	case ".tif", ".tiff":
		mimeType = "image/tiff"
	case ".jpg", ".jpeg", ".jpe":
		mimeType = "image/jpeg"
	case ".gif":
		mimeType = "image/gif"
	case ".ief":
		mimeType = "image/ief"
	case ".png":
		mimeType = "image/png"
	case ".dwg":
		mimeType = "image/vnd.dwg"
	case ".svg":
		mimeType = "image/svg+xml"
	case ".webp":
		mimeType = "image/webp"

	case ".aac":
		mimeType = "audio/aac"
	case ".mp1", ".mp2", ".mp3", ".mpg":
		mimeType = "audio/mpeg"
	case ".wma":
		mimeType = "audio/wma"
	case ".m4a":
		mimeType = "audio/mp4"
	case ".oga", ".ogg", ".opus", ".spx":
		mimeType = "audio/ogg"
	case ".mka":
		mimeType = "audio/x-matroska"
	case ".flac":
		mimeType = "audio/flac"

	case ".mp4", ".m4v":
		mimeType = "video/mp4"
	case ".3gp":
		mimeType = "video/3gp"
	case ".ogv":
		mimeType = "video/ogg"
	case ".webm":
		mimeType = "video/webm"
	case ".mov":
		mimeType = "video/quicktime"
	case ".mkv":
		mimeType = "video/x-matroska"
	case ".mk3d":
		mimeType = "video/x-matroska-3d"

	case ".apk":
		mimeType = "application/vnd.android.package-archive"
	case ".exe":
		mimeType = "application/exe"
	case ".doc", ".docx":
		mimeType = "application/msword"
	case ".xls", ".xlsx":
		mimeType = "application/vnd.ms-excel"
	case ".pdf":
		mimeType = "application/pdf"
	case ".rar":
		mimeType = "application/x-rar-compressed"
	case ".zip":
		mimeType = "application/zip"
	case ".ogx":
		mimeType = "application/ogg"

	default:
		mimeType = "application/octet-stream"
	}

	return mimeType
}

func GetTypeByFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	fileType := FileTypeOther

	switch ext {
	case ".bmp", ".tif", ".tiff", ".jpg", ".jpeg", ".jpe", ".ief", ".png", ".webp", ".svg":
		fileType = FileTypeImage

	case ".gif":
		fileType = FileTypeGif

	case ".aac", ".mp1", ".mp2", ".mp3", ".mpg", ".wma", ".m4a", ".oga", ".ogg", ".opus", ".spx", ".flac", ".mka":
		fileType = FileTypeAudio

	case ".doc", ".docx", ".xls", ".xlsx", ".pdf":
		fileType = FileTypeDocument

	case ".mp4", ".m4v", ".3gp", ".ogv", ".webm", ".mov", ".mkv", ".mk3d":
		fileType = FileTypeVideo
	}

	return fileType
}
