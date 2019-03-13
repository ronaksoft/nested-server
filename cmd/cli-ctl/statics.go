package main

import (
    "gopkg.in/yaml.v2"
    "log"
    "text/template"
    "fmt"
    "os"
    "io"
    "io/ioutil"
    "crypto/rsa"
    "encoding/pem"
    "crypto/x509"
    "strings"
    "regexp"
    "crypto/rand"
    "time"
    mrand "math/rand"
    "strconv"
    "crypto/md5"
    "math/big"
    "crypto/x509/pkix"
)

func LoadNestedConfig() *Config {
    cnf := new(Config)
    if yml, err := ioutil.ReadFile(pathConfigFile); err != nil {
        log.Println("Reading YAML file error: ", err.Error())
        return nil
    } else {
        if err := yaml.Unmarshal(yml, cnf); err != nil {
            log.Println("ReadConfig::Error::", err.Error())
            return nil
        }
    }
    return cnf
}

func UpdateYamlFiles(c *Config) bool {
    if c.EnabledServices.Mongo {
        serviceName := "mongo"
        ExecuteTemplate(serviceName, c)
        os.MkdirAll(fmt.Sprintf("%s/%s/.vol", pathYMLsDir, serviceName), os.ModeDir|os.ModePerm)
        CopyFile(fmt.Sprintf("%s/mongo.config.yml", pathTemplatesDir), fmt.Sprintf("%s/%s/config.yml", pathYMLsDir, serviceName))
        CopyFile(fmt.Sprintf("%s/%s.pem", pathCertsDir, serviceName), fmt.Sprintf("%s/%s/certs/%s.pem", pathYMLsDir, serviceName, serviceName))
    }
    if c.EnabledServices.Redis {
        serviceName := "redis"
        ExecuteTemplate(serviceName, c)
        os.MkdirAll(fmt.Sprintf("%s/%s/.vol", pathYMLsDir, serviceName), os.ModeDir|os.ModePerm)
        CopyFile(fmt.Sprintf("%s/%s.crt", pathCertsDir, serviceName), fmt.Sprintf("%s/%s/certs/%s.crt", pathYMLsDir, serviceName, serviceName))
        CopyFile(fmt.Sprintf("%s/%s.key", pathCertsDir, serviceName), fmt.Sprintf("%s/%s/certs/%s.key", pathYMLsDir, serviceName, serviceName))
    }
    if c.EnabledServices.Arsaces {
        serviceName := "arsaces"
        ExecuteTemplate(serviceName, c)
    }
    if c.EnabledServices.Cyrus {
        serviceName := "cyrus"
        ExecuteTemplate(serviceName, c)
        CopyFile(fmt.Sprintf("%s/%s.crt", pathCertsDir, serviceName), fmt.Sprintf("%s/%s/certs/%s.crt", pathYMLsDir, serviceName, serviceName))
        CopyFile(fmt.Sprintf("%s/%s.key", pathCertsDir, serviceName), fmt.Sprintf("%s/%s/certs/%s.key", pathYMLsDir, serviceName, serviceName))
        if _, err := os.Stat(fmt.Sprintf("%s/%s/domainkeys/dkim.private", pathYMLsDir, serviceName)); os.IsNotExist(err) {
            key, text := CreateDKIMKeys()
            fmt.Println("DKIM DNS:")
            fmt.Println(text)
            ioutil.WriteFile(
                fmt.Sprintf("%s/%s/domainkeys/dkim.private", pathYMLsDir, serviceName),
                []byte(key),
                0644,
            )
        }
    }
    if c.EnabledServices.Web {
        serviceName := "web"
        ExecuteTemplate(serviceName, c)
        CopyFile(fmt.Sprintf("%s/%s.crt", pathCertsDir, serviceName), fmt.Sprintf("%s/%s/certs/%s.crt", pathYMLsDir, serviceName, serviceName))
        CopyFile(fmt.Sprintf("%s/%s.key", pathCertsDir, serviceName), fmt.Sprintf("%s/%s/certs/%s.key", pathYMLsDir, serviceName, serviceName))
    }
    return true
}

func ExecuteTemplate(serviceName string, c *Config) bool {
    os.MkdirAll(fmt.Sprintf("%s/%s", pathYMLsDir, serviceName), os.ModeDir|os.ModePerm)
    b, _ := ioutil.ReadFile(fmt.Sprintf("%s/%s.yml", pathTemplatesDir, serviceName))
    t, _ := template.New(serviceName).Parse(string(b))
    if f, err := os.OpenFile(fmt.Sprintf("%s/%s/docker-compose.yml", pathYMLsDir, serviceName), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0750); err != nil {
        f.Close()
        log.Println("UpdateYamlFiles::Error 1::", err.Error())
        return false
    } else if err = t.Execute(f, c); err != nil {
        f.Close()
        log.Println(err.Error())
        CopyFile(fmt.Sprintf("%s/%s.yml", pathTemplatesDir, serviceName), fmt.Sprintf("%s/%s/docker-compose.yml", pathYMLsDir, serviceName))
    }
    os.MkdirAll(fmt.Sprintf("%s/%s/certs", pathYMLsDir, serviceName), os.ModeDir|os.ModePerm)
    CopyFile(fmt.Sprintf("%s/%s.crt", pathCertsDir, serviceName), fmt.Sprintf("%s/%s/certs/", pathYMLsDir, serviceName))
    CopyFile(fmt.Sprintf("%s/%s.key", pathCertsDir, serviceName), fmt.Sprintf("%s/%s/certs/", pathYMLsDir, serviceName))
    return true
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
