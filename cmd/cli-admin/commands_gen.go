package main

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"log"
	"math/big"
	mrand "math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/*
   Creation Time: 2021 - Aug - 07
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

func init() {
	RootCmd.AddCommand(GenCmd)
	GenCmd.AddCommand(GenDKIMCmd, GenRandomKey)
}

var GenCmd = &cobra.Command{
	Use:   "gen",
	Short: "generator for certificate, dkim etc.",
	Run:   func(cmd *cobra.Command, args []string) {},
}

// GenDKIMCmd Generates DKIM key-pairs
var GenDKIMCmd = &cobra.Command{
	Use:   "dkim",
	Short: "generate dkim public and private key",
	Run: func(cmd *cobra.Command, args []string) {
		priv, pub := CreateDKIMKeys()
		fmt.Println("DKIM Private Key:\r\n", priv)
		fmt.Println("DNS TEXT:\r\n", pub)
	},
}

// GenRandomKey Generate random key:
//	--length
var GenRandomKey = &cobra.Command{
	Use:   "key",
	Short: "generate random key",
	Run: func(cmd *cobra.Command, args []string) {
		length, _ := strconv.Atoi(cmd.Flag("length").Value.String())
		key := RandomID(length)
		fmt.Println(key)
	},
}

// CreateDKIMKeys produces a pair of public and private keys
func CreateDKIMKeys() (priv, pub string) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	priv = string(pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
		},
	))
	b, _ := x509.MarshalPKIXPublicKey(publicKey)
	p := string(pem.EncodeToMemory(
		&pem.Block{
			Type:  "",
			Bytes: b,
		},
	))
	p = strings.TrimSpace(p)
	p = strings.TrimPrefix(p, "-----BEGIN -----")
	p = strings.TrimSuffix(p, "-----END -----")

	r, _ := regexp.Compile("\n")
	p = r.ReplaceAllString(p, "")
	pub = fmt.Sprintf("v=DKIM1;t=s;n=core;p=%s", p)
	return
}

// CreateSelfSignedCertificate creates certificate for 1 year
// Use this function to create certificate for ServerAuth or ClientAuth required by other nested services
func CreateSelfSignedCertificate(commonName string, orgUnits []string) *SSLCertificate {
	cert := new(SSLCertificate)
	cert.CommonName = commonName
	cert.OrganizationalUnits = orgUnits
	extKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageAny}

	certTemplate := new(x509.Certificate)
	certTemplate.IsCA = false
	certTemplate.BasicConstraintsValid = true
	certTemplate.SubjectKeyId = md5.New().Sum([]byte(commonName))
	certTemplate.SerialNumber = big.NewInt(time.Now().UnixNano())
	certTemplate.Subject = pkix.Name{
		Country:            []string{"Iran", "Poland"},
		Organization:       []string{"Ronak Software Group"},
		OrganizationalUnit: orgUnits,
		CommonName:         commonName,
	}
	certTemplate.NotBefore = time.Now()
	certTemplate.NotAfter = time.Now().AddDate(1, 0, 0)

	certTemplate.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	certTemplate.ExtKeyUsage = extKeyUsage

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	if c, err := x509.CreateCertificate(rand.Reader, certTemplate, certTemplate, publicKey, privateKey); err != nil {
		log.Println("CreateCertificate::Error::", err.Error())
	} else {
		cert.Key = pem.EncodeToMemory(
			&pem.Block{
				Type:  "RSA PRIVATE KEY",
				Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
			},
		)
		cert.Certificate = pem.EncodeToMemory(
			&pem.Block{
				Type:  "CERTIFICATE",
				Bytes: c,
			},
		)
	}
	return cert
}

// RandomID generates a random string with length of 'n'
func RandomID(n int) string {
	mrand.Seed(time.Now().UnixNano())
	ts := strings.ToUpper(strconv.FormatInt(time.Now().UnixNano(), 16))
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const size = 36
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[mrand.Intn(size)]
	}
	return fmt.Sprintf("%s%s", ts, string(b))
}

func CopyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	if err != nil {
		return
	}
	if !sfi.Mode().IsRegular() {
		// cannot copy non-regular files (e.g., directories,
		// symlinks, devices, etc.)
		return fmt.Errorf("CopyFile: non-regular source file %s (%q)", sfi.Name(), sfi.Mode().String())
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			return
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			return fmt.Errorf("CopyFile: non-regular destination file %s (%q)", dfi.Name(), dfi.Mode().String())
		}
		if os.SameFile(sfi, dfi) {
			return
		}
	}
	if err = os.Link(src, dst); err == nil {
		return
	}
	err = copyFileContents(src, dst)
	return
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}

type SSLCertificate struct {
	CommonName          string   `bson:"cn" json:"cn"`
	OrganizationalUnits []string `bson:"ou" json:"ou"`
	Key                 []byte   `bson:"key" json:"key"`
	Certificate         []byte   `bson:"cert" json:"cert"`
}
