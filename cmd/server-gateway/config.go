package main

import (
	"gopkg.in/fzerorubigd/onion.v3"
	"gopkg.in/fzerorubigd/onion.v3/extraenv"
)

const (
	CONF_DOMAINS              = "DOMAINS" // comma separated
	CONF_SENDER_DOMAIN        = "SENDER_DOMAIN"
	CONF_BUNDLE_ID            = "BUNDLE_ID"
	CONF_BIND_ADDRESS         = "BIND_ADDRESS"
	CONF_TLS_KEY_FILE         = "TLS_KEY_FILE"
	CONF_TLS_CERT_FILE        = "TLS_CERT_FILE"
	CONF_JOB_ADDRESS          = "JOB_ADDRESS"
	CONF_MONGO_TLS            = "MONGO_TLS"
	CONF_MONGO_DSN            = "MONGO_DSN"
	CONF_REDIS_DSN            = "REDIS_DSN"
	CONF_DEBUG_LEVEL          = "DEBUG_LEVEL"
	CONF_ADP_MESSAGE_URL      = "ADP_MESSAGE_URL"
	CONF_ADP_USERNAME         = "ADP_USERNAME"
	CONF_ADP_PASSWORD         = "ADP_PASSWORD"
	CONF_MONITOR_ACCESS_TOKEN = "MONITOR_ACCESS_TOKEN"
	CONF_FILE_SYSTEM_TOKEN    = "FILE_SYSTEM_KEY"
	CONF_SMTP_USER            = "SMTP_USER"
	CONF_SMTP_PASS            = "SMTP_PASS"
	CONF_SMTP_HOST            = "SMTP_HOST"
	CONF_SMTP_PORT            = "SMTP_PORT"
	CONF_CYRUS_URL            = "CYRUS_URL"
	CONF_INSTANCE_ID          = "INSTANCE_ID"
)

func readConfig() *onion.Onion {
	dl := onion.NewDefaultLayer()

	dl.SetDefault(CONF_DOMAINS, "nested.me") // comma separated
	dl.SetDefault(CONF_SENDER_DOMAIN, "nested.me")
	dl.SetDefault(CONF_BUNDLE_ID, "CYRUS.001")
	dl.SetDefault(CONF_BIND_ADDRESS, "0.0.0.0:8080")
	dl.SetDefault(CONF_CYRUS_URL, "http://storage.xerxes.nst")

	// InstanceID
	dl.SetDefault(CONF_INSTANCE_ID, "")

	// Security
	dl.SetDefault(CONF_TLS_KEY_FILE, "")
	dl.SetDefault(CONF_TLS_CERT_FILE, "")
	dl.SetDefault(CONF_JOB_ADDRESS, "nats://localhost:4222")

	// Database (MongoDB)
	dl.SetDefault(CONF_MONGO_TLS, true)
	dl.SetDefault(CONF_MONGO_DSN, "localhost:27017")

	// Cache (Redis)
	dl.SetDefault(CONF_REDIS_DSN, "localhost:6379")

	// Debugging
	dl.SetDefault(CONF_DEBUG_LEVEL, 2)

	// ADP Configs
	dl.SetDefault(CONF_ADP_USERNAME, "ronak")
	dl.SetDefault(CONF_ADP_PASSWORD, "E2e2374k19743")
	dl.SetDefault(CONF_ADP_MESSAGE_URL, "https://ws.adpdigital.com/url/send")

	// SMTP
	dl.SetDefault(CONF_SMTP_HOST, "mta")
	dl.SetDefault(CONF_SMTP_PORT, 25)
	dl.SetDefault(CONF_SMTP_USER, "user")
	dl.SetDefault(CONF_SMTP_PASS, "pa$$word")

	// Extra Configs
	dl.SetDefault(CONF_MONITOR_ACCESS_TOKEN, "!@NES##monitor##TED@!")
	dl.SetDefault(CONF_FILE_SYSTEM_TOKEN, "testKey")

	cfg := onion.New()
	cfg.AddLayer(dl)
	cfg.AddLazyLayer(extraenv.NewExtraEnvLayer("NST"))

	return cfg
}
