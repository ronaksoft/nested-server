package main

import (
	"gopkg.in/fzerorubigd/onion.v3"
	"gopkg.in/fzerorubigd/onion.v3/extraenv"
)

const (
	ConfDomains            = "DOMAINS" // comma separated
	ConfSenderDomain       = "SENDER_DOMAIN"
	ConfBundleId           = "BUNDLE_ID"
	ConfBindAddress        = "BIND_ADDRESS"
	ConfTlsKeyFile         = "TLS_KEY_FILE"
	ConfTlsCertFile        = "TLS_CERT_FILE"
	ConfJobAddress         = "JOB_ADDRESS"
	ConfMongoTls           = "MONGO_TLS"
	ConfMongoDsn           = "MONGO_DSN"
	ConfRedisDsn           = "REDIS_DSN"
	ConfDebugLevel         = "DEBUG_LEVEL"
	ConfAdpMessageUrl      = "ADP_MESSAGE_URL"
	ConfAdpUsername        = "ADP_USERNAME"
	ConfAdpPassword        = "ADP_PASSWORD"
	ConfMonitorAccessToken = "MONITOR_ACCESS_TOKEN"
	ConfFileSystemToken    = "FILE_SYSTEM_KEY"
	ConfSmtpUser           = "SMTP_USER"
	ConfSmtpPass           = "SMTP_PASS"
	ConfSmtpHost           = "SMTP_HOST"
	ConfSmtpPort           = "SMTP_PORT"
	ConfCyrusUrl           = "CYRUS_URL"
	ConfInstanceId         = "INSTANCE_ID"
)

func readConfig() *onion.Onion {
	dl := onion.NewDefaultLayer()

	_ = dl.SetDefault(ConfDomains, "nested.me") // comma separated
	_ = dl.SetDefault(ConfSenderDomain, "nested.me")
	_ = dl.SetDefault(ConfBundleId, "CYRUS.001")
	_ = dl.SetDefault(ConfBindAddress, "0.0.0.0:8080")
	_ = dl.SetDefault(ConfCyrusUrl, "http://storage.xerxes.nst")

	// InstanceID
	_ = dl.SetDefault(ConfInstanceId, "")

	// Security
	_ = dl.SetDefault(ConfTlsKeyFile, "")
	_ = dl.SetDefault(ConfTlsCertFile, "")
	_ = dl.SetDefault(ConfJobAddress, "nats://localhost:4222")

	// Database (MongoDB)
	_ = dl.SetDefault(ConfMongoTls, true)
	_ = dl.SetDefault(ConfMongoDsn, "localhost:27017")

	// Cache (Redis)
	_ = dl.SetDefault(ConfRedisDsn, "localhost:6379")

	// Debugging
	_ = dl.SetDefault(ConfDebugLevel, 2)

	// ADP Configs
	_ = dl.SetDefault(ConfAdpUsername, "ronak")
	_ = dl.SetDefault(ConfAdpPassword, "E2e2374k19743")
	_ = dl.SetDefault(ConfAdpMessageUrl, "https://ws.adpdigital.com/url/send")

	// SMTP
	_ = dl.SetDefault(ConfSmtpHost, "mta")
	_ = dl.SetDefault(ConfSmtpPort, 25)
	_ = dl.SetDefault(ConfSmtpUser, "user")
	_ = dl.SetDefault(ConfSmtpPass, "pa$$word")

	// Extra Configs
	_ = dl.SetDefault(ConfMonitorAccessToken, "!@NES##monitor##TED@!")
	_ = dl.SetDefault(ConfFileSystemToken, "testKey")

	cfg := onion.New()
	_ = cfg.AddLayer(dl)
	cfg.AddLazyLayer(extraenv.NewExtraEnvLayer("NST"))

	return cfg
}
