package main

import (
    "github.com/spf13/cobra"
    "fmt"
    "strconv"
    "io/ioutil"
    "os"
)

var GenCmd = &cobra.Command{
    Use:   "gen",
    Short: "generator for certificate, dkim etc.",
    Run:   func(cmd *cobra.Command, args []string) {},
}

// Generates DKIM key-pairs
var GenDKIMCmd = &cobra.Command{
    Use:   "dkim",
    Short: "generate dkim public and private key",
    Run: func(cmd *cobra.Command, args []string) {
        priv, pub := CreateDKIMKeys()
        fmt.Println("DKIM Private Key:\r\n", priv)
        fmt.Println("DNS TEXT:\r\n", pub)
    },
}

// Generate random key:
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

// Generate Self-signed certificate:
//	--cn			common name
//	--ou			organization
//	--name			filename
//	--singleFile
var GenSelfSignedCertificate = &cobra.Command{
    Use:   "selfSignedCertificate",
    Short: "Generates a self-signed certificate",
    Run: func(cmd *cobra.Command, args []string) {
        commonName := cmd.Flag("cn").Value.String()
        orgUnit := cmd.Flag("ou").Value.String()
        filename := cmd.Flag("name").Value.String()
        singleFile := cmd.Flag("singleFile").Changed
        if ssl := CreateSelfSignedCertificate(commonName, []string{orgUnit}); ssl != nil {
            if singleFile {
                ioutil.WriteFile(fmt.Sprintf("%s/%s.pem", pathCertsDir, filename), append(ssl.Certificate, ssl.Key...), os.ModePerm)
            } else {
                ioutil.WriteFile(fmt.Sprintf("%s/%s.crt", pathCertsDir, filename), ssl.Certificate, os.ModePerm)
                ioutil.WriteFile(fmt.Sprintf("%s/%s.key", pathCertsDir, filename), ssl.Key, os.ModePerm)
            }
            fmt.Println(fmt.Sprintf("Certificate created succesfuly in cert directory (%s)", pathCertsDir))
        }
    },
}

func init() {
    RootCmd.AddCommand(GenCmd)
    GenCmd.AddCommand(GenDKIMCmd, GenRandomKey, GenSelfSignedCertificate)

    GenRandomKey.Flags().Int("length", 32, "length of the key")

    GenSelfSignedCertificate.Flags().String("ou", "Nested Service", "Organization Unit")
    GenSelfSignedCertificate.Flags().String("cn", "*.nested.me", "Common Name")
    GenSelfSignedCertificate.Flags().String("name", "selfsigned", "Output will be: name.crt, name.key")
    GenSelfSignedCertificate.Flags().Bool("singleFile", false, "Put key and certificate in single file")
}
