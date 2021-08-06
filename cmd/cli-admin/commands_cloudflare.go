package main

import (
	"fmt"
	"github.com/cloudflare/cloudflare-go"
	"github.com/kr/pretty"
	"github.com/spf13/cobra"
	"log"
)

var CFCmd = &cobra.Command{
	Use:   "cf",
	Short: "Cloudflare APIs",
}

var CFCreateARecordCmd = &cobra.Command{
	Use: "createARecord",
	Run: func(cmd *cobra.Command, args []string) {

		subDomain := cmd.Flag("subDomain").Value.String()
		ip := cmd.Flag("ip").Value.String()

		cfAPI, _ := cloudflare.New(_Config.GetString("CF_GLOBAL_API_KEY"), _Config.GetString("CF_EMAIL_ADDR"))
		zoneID, _ := cfAPI.ZoneIDByName(_Config.GetString("DOMAIN"))
		dnsRecord := cloudflare.DNSRecord{
			Type:    "A",
			Name:    fmt.Sprintf("%s.%s", subDomain, _Config.GetString("DOMAIN")),
			Content: ip,
		}
		if dnsResponse, err := cfAPI.CreateDNSRecord(zoneID, dnsRecord); err != nil {
			log.Println("CreateDNSRecord::Error::", err.Error())
		} else {
			pretty.Println(dnsResponse)
		}
	},
}

var CFCreateMXRecordCmd = &cobra.Command{
	Use: "createMXRecord",
	Run: func(cmd *cobra.Command, args []string) {
		subDomain := cmd.Flag("subDomain").Value.String()

		cfAPI, _ := cloudflare.New(_Config.GetString("CF_GLOBAL_API_KEY"), _Config.GetString("CF_EMAIL_ADDR"))
		zoneID, _ := cfAPI.ZoneIDByName(_Config.GetString("DOMAIN"))
		dnsRecord := cloudflare.DNSRecord{
			Type:     "MX",
			Name:     fmt.Sprintf("%s.%s", subDomain, _Config.GetString("DOMAIN")),
			Content:  fmt.Sprintf("%s.%s", subDomain, _Config.GetString("DOMAIN")),
			Priority: 10,
		}
		if dnsResponse, err := cfAPI.CreateDNSRecord(zoneID, dnsRecord); err != nil {
			log.Println("CreateDNSRecord::Error::", err.Error())
		} else {
			pretty.Println(dnsResponse)
		}
	},
}

var CFCreateDKIMRecordCmd = &cobra.Command{
	Use: "createDKIMRecord",
	Run: func(cmd *cobra.Command, args []string) {
		value := cmd.Flag("value").Value.String()
		subDomain := cmd.Flag("subDomain").Value.String()
		cfAPI, _ := cloudflare.New(
			_Config.GetString("CF_GLOBAL_API_KEY"),
			_Config.GetString("CF_EMAIL_ADDR"),
		)
		zoneID, _ := cfAPI.ZoneIDByName(_Config.GetString("DOMAIN"))
		dnsRecord := cloudflare.DNSRecord{
			Type:    "TXT",
			Name:    fmt.Sprintf("mail._domainkey.%s.%s", subDomain, _Config.GetString("DOMAIN")),
			Content: fmt.Sprintf("%s", value),
		}
		if dnsResponse, err := cfAPI.CreateDNSRecord(zoneID, dnsRecord); err != nil {
			log.Println("CreateDNSRecord::Error::", err.Error())
		} else {
			pretty.Println(dnsResponse)
		}
	},
}

var CFCreateNestedRecordCmd = &cobra.Command{
	Use: "createNestedRecord",
	Run: func(cmd *cobra.Command, args []string) {
		subDomain := cmd.Flag("subDomain").Value.String()
		cfAPI, _ := cloudflare.New(
			_Config.GetString("CF_GLOBAL_API_KEY"),
			_Config.GetString("CF_EMAIL_ADDR"),
		)
		zoneID, _ := cfAPI.ZoneIDByName(_Config.GetString("DOMAIN"))

		subDomainAddr := fmt.Sprintf("%s.%s", subDomain, _Config.GetString("DOMAIN"))
		cyrusHttpRecord := fmt.Sprintf("cyrus:https:81:%s", subDomainAddr)
		cyrusWsRecord := fmt.Sprintf("cyrus:wss:81:%s", subDomainAddr)
		XerxesRecord := fmt.Sprintf("xerxes:https:83:%s", subDomainAddr)
		dnsRecord := cloudflare.DNSRecord{
			Type:    "TXT",
			Name:    fmt.Sprintf("_nested.%s.%s", subDomain, _Config.GetString("DOMAIN")),
			Content: fmt.Sprintf("%s;%s;%s", cyrusHttpRecord, cyrusWsRecord, XerxesRecord),
		}
		if dnsResponse, err := cfAPI.CreateDNSRecord(zoneID, dnsRecord); err != nil {
			log.Println("CreateDNSRecord::Error::", err.Error())
		} else {
			pretty.Println(dnsResponse)
		}
	},
}

var CFListDnsRecordsCmd = &cobra.Command{
	Use: "listNestedRecords",
	Run: func(cmd *cobra.Command, args []string) {
		cfAPI, _ := cloudflare.New(
			_Config.GetString("CF_GLOBAL_API_KEY"),
			_Config.GetString("CF_EMAIL_ADDR"),
		)
		zoneID, _ := cfAPI.ZoneIDByName(_Config.GetString("DOMAIN"))
		recordName := cmd.Flag("recordName").Value.String()
		if dnsRecords, err := cfAPI.DNSRecords(zoneID, cloudflare.DNSRecord{
			Name: fmt.Sprintf("%s.nested.me", recordName),
		}); err != nil {
			log.Println("CFListDnsRecordsCmd::Error::", err.Error())
		} else {
			for _, dnsRecord := range dnsRecords {
				fmt.Println("=========")
				fmt.Println("ID:", dnsRecord.ID)
				fmt.Println("EventType:", dnsRecord.Type)
				fmt.Println("Name:", dnsRecord.Name)
				fmt.Println("Content:", dnsRecord.Content)
				fmt.Println("=========")
			}
		}
	},
}

var CFRemoveNestedRecordCmd = &cobra.Command{
	Use: "removeNestedRecord",
	Run: func(cmd *cobra.Command, args []string) {
		cfAPI, _ := cloudflare.New(_Config.GetString("CF_GLOBAL_API_KEY"), _Config.GetString("CF_EMAIL_ADDR"))
		zoneID, _ := cfAPI.ZoneIDByName(_Config.GetString("DOMAIN"))
		recordID := cmd.Flag("recordID").Value.String()
		if err := cfAPI.DeleteDNSRecord(zoneID, recordID); err != nil {
			log.Println("CFRemoveNestedRecordCmd::Error::", err.Error())
		}
	},
}

var CFInstallNestedCmd = &cobra.Command{
	Use: "installNested",
	Run: func(cmd *cobra.Command, args []string) {
		subDomain := cmd.Flag("subDomain").Value.String()
		ip := cmd.Flag("ip").Value.String()
		cfAPI, _ := cloudflare.New(
			_Config.GetString("CF_GLOBAL_API_KEY"),
			_Config.GetString("CF_EMAIL_ADDR"),
		)
		zoneID, _ := cfAPI.ZoneIDByName(_Config.GetString("DOMAIN"))

		// Create A Record
		ARecord := cloudflare.DNSRecord{
			Type:    "A",
			Name:    fmt.Sprintf("%s.%s", subDomain, _Config.GetString("DOMAIN")),
			Content: ip,
		}
		if dnsResponse, err := cfAPI.CreateDNSRecord(zoneID, ARecord); err != nil {
			log.Println("CFInstallNestedCmd::Error::", err.Error())
		} else {
			pretty.Println(dnsResponse)
		}

		// Create MX Record
		MXRecord := cloudflare.DNSRecord{
			Type:     "MX",
			Name:     fmt.Sprintf("%s.%s", subDomain, _Config.GetString("DOMAIN")),
			Content:  fmt.Sprintf("%s.%s", subDomain, _Config.GetString("DOMAIN")),
			Priority: 10,
		}
		if dnsResponse, err := cfAPI.CreateDNSRecord(zoneID, MXRecord); err != nil {
			log.Println("CFInstallNestedCmd::Error::", err.Error())
		} else {
			pretty.Println(dnsResponse)
		}

		// Create TXT Record
		subDomainAddr := fmt.Sprintf("%s.%s", subDomain, _Config.GetString("DOMAIN"))
		cyrusHttpRecord := fmt.Sprintf("cyrus:https:81:%s", subDomainAddr)
		cyrusWsRecord := fmt.Sprintf("cyrus:wss:81:%s", subDomainAddr)
		XerxesRecord := fmt.Sprintf("xerxes:https:83:%s", subDomainAddr)
		txtRecord := cloudflare.DNSRecord{
			Type:    "TXT",
			Name:    fmt.Sprintf("_nested.%s.%s", subDomain, _Config.GetString("DOMAIN")),
			Content: fmt.Sprintf("%s;%s;%s", cyrusHttpRecord, cyrusWsRecord, XerxesRecord),
		}
		if dnsResponse, err := cfAPI.CreateDNSRecord(zoneID, txtRecord); err != nil {
			log.Println("CFInstallNestedCmd::Error::", err.Error())
		} else {
			pretty.Println(dnsResponse)
		}

	},
}

var CFUninstallNestedCmd = &cobra.Command{
	Use: "uninstallNested",
	Run: func(cmd *cobra.Command, args []string) {
		subDomain := cmd.Flag("subDomain").Value.String()
		cfAPI, _ := cloudflare.New(
			_Config.GetString("CF_GLOBAL_API_KEY"),
			_Config.GetString("CF_EMAIL_ADDR"),
		)
		zoneID, _ := cfAPI.ZoneIDByName(_Config.GetString("DOMAIN"))

		if dnsRecords, err := cfAPI.DNSRecords(zoneID, cloudflare.DNSRecord{
			Name: fmt.Sprintf("%s.%s", subDomain, _Config.GetString("DOMAIN")),
		}); err != nil {
			log.Println("CFUninstallNested::Error::", err.Error())
		} else {
			for _, dnsRecord := range dnsRecords {
				cfAPI.DeleteDNSRecord(zoneID, dnsRecord.ID)
				fmt.Println(fmt.Sprintf("DNS (%s:%s) Removed.", dnsRecord.Type, dnsRecord.Name))
			}
		}
		if dnsRecords, err := cfAPI.DNSRecords(zoneID, cloudflare.DNSRecord{
			Name: fmt.Sprintf("_nested.%s.%s", subDomain, _Config.GetString("DOMAIN")),
		}); err != nil {
			log.Println("CFUninstallNested::Error::", err.Error())
		} else {
			for _, dnsRecord := range dnsRecords {
				cfAPI.DeleteDNSRecord(zoneID, dnsRecord.ID)
				fmt.Println(fmt.Sprintf("DNS (%s:%s) Removed.", dnsRecord.Type, dnsRecord.Name))
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(CFCmd)
	CFCmd.AddCommand(
		CFCreateMXRecordCmd, CFCreateNestedRecordCmd, CFCreateARecordCmd, CFCreateDKIMRecordCmd,
		CFListDnsRecordsCmd, CFRemoveNestedRecordCmd, CFInstallNestedCmd, CFUninstallNestedCmd,
	)
	CFCmd.PersistentFlags().String("domain", "", "default is : nested.me")
	CFCmd.PersistentFlags().String("subDomain", "", "<sub-domain>.<domain>")
	CFCmd.PersistentFlags().String("recordName", "", "")
	CFCmd.PersistentFlags().String("recordID", "", "")

	CFCreateARecordCmd.Flags().String("ip", "", "enter ip")

	CFCreateDKIMRecordCmd.Flags().String("value", "", "dkim value")
}
