package main

import (
	"bufio"
	"crypto/sha512"
	"encoding/json"
	"fmt"
	"git.ronaksoft.com/nested/server/model"
	"github.com/spf13/cobra"
	"os"
	"strconv"
	"strings"
	"time"
)

var LicenseCmd = &cobra.Command{
	Use:   "license",
	Short: "License Manager",
}

var LicenseGenerateCmd = &cobra.Command{
	Use:   "gen",
	Short: "Generate License",
	Run: func(cmd *cobra.Command, args []string) {
		var licenseExpireDate string

		license := nested.License{}
		reader := bufio.NewReader(os.Stdin)

		// Ask for Owner's Name
		fmt.Print("Name: ")
		license.OwnerName, _ = reader.ReadString('\n')
		license.OwnerName = strings.TrimSpace(license.OwnerName)

		// Ask for Owner's Organization
		fmt.Print("Organization: ")
		license.OwnerOrganization, _ = reader.ReadString('\n')
		license.OwnerOrganization = strings.TrimSpace(license.OwnerOrganization)

		// Ask for Owner's Email address
		fmt.Print("Email Address: ")
		license.OwnerEmail, _ = reader.ReadString('\n')
		license.OwnerEmail = strings.TrimSpace(license.OwnerEmail)

		// Ask for License Expire Date
		fmt.Print("License Expire Date (format: YYYY-MM-DD): ")
		licenseExpireDate, _ = reader.ReadString('\n')
		licenseExpireDate = strings.TrimSpace(licenseExpireDate)
		licenseYear, _ := strconv.Atoi(licenseExpireDate[:4])
		licenseMonth, _ := strconv.Atoi(licenseExpireDate[5:7])
		licenseDay, _ := strconv.Atoi(licenseExpireDate[8:10])
		license.ExpireDate = uint64(time.Date(licenseYear, time.Month(licenseMonth), licenseDay, 0, 0, 0, 0, time.Local).Unix() * 1000)

		// Ask for Max Active Users
		fmt.Print("Max Active Users: ")
		fmt.Scan(&license.MaxActiveUsers)

		license.LicenseID = nested.RandomID(64)
		license.Signature = Sha512([]byte(fmt.Sprintf("%s%d%d", license.LicenseID, license.ExpireDate, license.MaxActiveUsers)))

		fmt.Println("========================================")
		fmt.Println("#### Nested License")
		fmt.Println("========================================")
		fmt.Println("ID:", license.LicenseID)
		fmt.Println("Signature:", license.Signature)
		fmt.Println("Name:", license.OwnerName)
		fmt.Println("Email:", license.OwnerEmail)
		fmt.Println("Expire Date:", time.Unix(int64(license.ExpireDate/1000), 0))
		fmt.Println("Active Users:", license.MaxActiveUsers)
		fmt.Println("========================================")

		b, _ := json.Marshal(license)
		licenseFile := nested.Encrypt(nested.LICENSE_ENCRYPT_KEY, string(b))
		fmt.Println("License Key: (Copy & paste the generated code below)")
		fmt.Println(licenseFile)
	},
}

func init() {
	RootCmd.AddCommand(LicenseCmd)
	LicenseCmd.AddCommand(LicenseGenerateCmd)
}

func Sha512(in []byte) []byte {
	_funcName := "FrontServer::Sha512"
	h := sha512.New()
	if _, err := h.Write(in); err != nil {
		fmt.Println(_funcName, err.Error())
	}
	return h.Sum(nil)
}
