package main

import (
    "strings"
    "os/exec"
    "fmt"
    "github.com/fatih/structs"
    "errors"
    "time"
)

type SSLCertificate struct {
    CommonName          string   `bson:"cn" json:"cn"`
    OrganizationalUnits []string `bson:"ou" json:"ou"`
    Key                 []byte   `bson:"key" json:"key"`
    Certificate         []byte   `bson:"cert" json:"cert"`
}

type ArsacesConfig struct {
    Host        string `yaml:"Conn"`
    Port        string `yaml:"Port"`
    PortExposed bool   `yaml:"PortExposed"`
}

type CyrusConfig struct {
    BundleID       string `yaml:"BundleID"`
    Host           string `yaml:"Conn"`
    Port           int    `yaml:"Port"`
    Secure         bool   `yaml:"Secure"`
    DebugLevel     int    `yaml:"DebugLevel"`
    WebappBaseUrl  string `yaml:"WebappBaseUrl"`
    SMTPUser       string `yaml:"SMTPUser"`
    SMTPPass       string `yaml:"SMTPPass"`
    SMTPPrivateKey string `yaml:"SMTPPrivateKey"`
    SMTPPublicKey  string `yaml:"SMTPPublicKey"`
    CyrusUrl       string `yaml:"CyrusUrl"`
}

type MongoConfig struct {
    DSN             string `yaml:"DSN"`
    Port            string `yaml:"Port"`
    PortExposed     bool   `yaml:"PortExposed"`
    FileDSN         string `yaml:"FileDSN"`
    FilePort        int    `yaml:"FilePort"`
    FilePortExposed bool   `yaml:"FilePortExposed"`
}

type RedisConfig struct {
    Host        string `yaml:"Conn"`
    Port        int    `yaml:"Port"`
    PortExposed bool   `yaml:"PortExposed"`
}

type WebappConfig struct {
    AppPort                  int    `yaml:"AppPort"`
    DisableFcm               bool   `yaml:"DisableFcm"`
    DefaultCyrusHttpUrl      string `yaml:"DefaultCyrusHttpUrl"`
    DefaultCyrusWebsocketUrl string `yaml:"DefaultCyrusWebsocketUrl"`
}

type EnabledServices struct {
    Cyrus   bool `yaml:"Cyrus"`
    Arsaces bool `yaml:"Arsaces"`
    Mongo   bool `yaml:"Mongo"`
    Redis   bool `yaml:"Redis"`
    Web     bool `yaml:"Web"`
}

type AutoGeneratedItems struct {
    GeneratedDKIMKey   string
    GeneratedDKIMText  string
    GeneratedXerxesKey string
}

type Config struct {
    Domain             string          `yaml:"Domain"`
    GoogleAPIKey       string          `yaml:"GoogleAPIKey"`
    ClientSideIP       string          `yaml:"ClientSideIP"`
    MongoDataDSN       string          `yaml:"MongoDataDSN"`
    MongoFileDSN       string          `yaml:"MongoFileDSN"`
    RedisCacheDSN      string          `yaml:"RedisCacheDSN"`
    ExternalJobUrl     string          `yaml:"ExternalJobUrl"`
    CyrusFileSystemKey string          `yaml:"CyrusFileSystemKey"`
    Arsaces            ArsacesConfig   `yaml:"Arsaces"`
    Cyrus              CyrusConfig     `yaml:"Cyrus"`
    Web                WebappConfig    `yaml:"Web"`
    Mongo              MongoConfig     `yaml:"Mongo"`
    Redis              RedisConfig     `yaml:"Redis"`
    EnabledServices    EnabledServices `yaml:"EnabledServices"`
}

func (c *Config) StartService(serviceName string) error {
    serviceName = strings.ToLower(serviceName)
    services := structs.Fields(c.EnabledServices)
    for _, service := range services {
        if strings.ToLower(service.Name()) == serviceName && service.Value().(bool) {
            fmt.Println("Run Service:", serviceName)
            dockerCommand := exec.Command(
                "docker-compose",
                "-f", fmt.Sprintf("%s/%s/docker-compose.yml", pathYMLsDir, serviceName), "up", "-d",
            )
            pipeOut, _ := dockerCommand.StdoutPipe()
            pipeErr, _ := dockerCommand.StderrPipe()
            if err := dockerCommand.Start(); err != nil {
                fmt.Println("Config::StartService::Error::", err.Error(), serviceName)
            }
            // Reading StrOut
            go func() {
                b := make([]byte, 1024, 1024)
                for {
                    if n, err := pipeOut.Read(b); err != nil {
                        break
                    } else {
                        fmt.Print(string(b[:n]))
                    }

                    time.Sleep(100 * time.Millisecond)
                }
            }()
            // Reading StdErr
            go func() {
                b := make([]byte, 1024, 1024)
                for {
                    if n, err := pipeErr.Read(b); err != nil {
                        break
                    } else {
                        fmt.Print(string(b[:n]))
                    }
                    time.Sleep(100 * time.Millisecond)
                }
            }()
            dockerCommand.Wait()
            return nil
        }
    }
    return errors.New("service does not exist or is not enabled")
}
func (c *Config) StopService(serviceName string) error {
    serviceName = strings.ToLower(serviceName)
    services := structs.Fields(c.EnabledServices)
    for _, service := range services {
        if strings.ToLower(service.Name()) == serviceName && service.Value().(bool) {
            fmt.Println("\nStop Service:", serviceName)
            dockerCommand := exec.Command("docker-compose", "-f", fmt.Sprintf("%s/%s/docker-compose.yml", pathYMLsDir, serviceName), "down")
            pipeOut, _ := dockerCommand.StdoutPipe()
            pipeErr, _ := dockerCommand.StderrPipe()
            if err := dockerCommand.Start(); err != nil {
                fmt.Println("Start Error:", err.Error())
            }
            // Reading StrOut
            go func() {
                b := make([]byte, 1024, 1024)
                for {
                    if n, err := pipeOut.Read(b); err != nil {
                        break
                    } else {
                        fmt.Print(string(b[:n]))
                    }

                    time.Sleep(100 * time.Millisecond)
                }
            }()
            // Reading StdErr
            go func() {
                b := make([]byte, 1024, 1024)
                for {
                    if n, err := pipeErr.Read(b); err != nil {
                        break
                    } else {
                        fmt.Print(string(b[:n]))
                    }
                    time.Sleep(100 * time.Millisecond)
                }
            }()
            dockerCommand.Wait()
            return nil
        }
    }
    return errors.New("Service does not exist or is not enabled")
}
func (c *Config) UpdateService(serviceName string) error {
    serviceName = strings.ToLower(serviceName)
    services := structs.Fields(c.EnabledServices)
    for _, service := range services {
        if strings.ToLower(service.Name()) == serviceName && service.Value().(bool) {
            fmt.Println("\nUpdate Service:", serviceName)
            dockerCommand := exec.Command("docker-compose", "-f", fmt.Sprintf("%s/%s/docker-compose.yml", pathYMLsDir, serviceName), "pull")
            pipeOut, _ := dockerCommand.StdoutPipe()
            pipeErr, _ := dockerCommand.StderrPipe()
            if err := dockerCommand.Start(); err != nil {
                fmt.Println("Start Error:", err.Error())
            }
            // Reading StrOut
            go func() {
                b := make([]byte, 1024, 1024)
                for {
                    if n, err := pipeOut.Read(b); err != nil {
                        break
                    } else {
                        fmt.Print(string(b[:n]))
                    }

                    time.Sleep(100 * time.Millisecond)
                }
            }()
            // Reading StdErr
            go func() {
                b := make([]byte, 1024, 1024)
                for {
                    if n, err := pipeErr.Read(b); err != nil {
                        break
                    } else {
                        fmt.Print(string(b[:n]))
                    }
                    time.Sleep(100 * time.Millisecond)
                }
            }()
            dockerCommand.Wait()
            return nil
        }
    }
    return errors.New("Service does not exist or is not enabled")
}
