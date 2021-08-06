package config

import (
	"gopkg.in/fzerorubigd/onion.v3"
	"gopkg.in/fzerorubigd/onion.v3/extraenv"
)

const (
	Domains            = "DOMAINS" // comma separated
	SenderDomain       = "SENDER_DOMAIN"
	BundleID           = "BUNDLE_ID"
	BindAddress        = "BIND_ADDRESS"
	CyrusURL           = "CYRUS_URL"
	TlsKeyFile         = "TLS_KEY_FILE"
	TlsCertFile        = "TLS_CERT_FILE"
	ConfJobAddress     = "JOB_ADDRESS"
	ConfMongoTls       = "MONGO_TLS"
	MongoDSN           = "MONGO_DSN"
	RedisDSN           = "REDIS_DSN"
	LogLevel           = "LOG_LEVEL"
	ADPMessageUrl      = "ADP_MESSAGE_URL"
	ADPUsername        = "ADP_USERNAME"
	ADPPassword        = "ADP_PASSWORD"
	MonitorAccessToken = "MONITOR_ACCESS_TOKEN"
	FileSystemKey      = "FILE_SYSTEM_KEY"
	SmtpUser           = "SMTP_USER"
	SmtpPass           = "SMTP_PASS"
	SmtpHost           = "SMTP_HOST"
	SmtpPort           = "SMTP_PORT"
	InstanceID         = "INSTANCE_ID"
	WebAppBaseURL      = "WEBAPP_BASE_URL"
	NestedDir          = "NESTED_DIR"
	PostfixCHRoot      = "POSTFIX_CHROOT"
	MailStoreSock      = "MAIL_STORE_SOCK"
)

var (
	_Onion *onion.Onion
)

func init() {
	dl := onion.NewDefaultLayer()
	_ = dl.SetDefault(PostfixCHRoot, "/var/spool/postfix")
	_ = dl.SetDefault(MailStoreSock, "private/nested-mail")

	_ = dl.SetDefault(NestedDir, "/ronak/nested")
	_ = dl.SetDefault(Domains, "nested.me") // comma separated
	_ = dl.SetDefault(SenderDomain, "nested.me")
	_ = dl.SetDefault(BundleID, "CYRUS.001")
	_ = dl.SetDefault(BindAddress, "0.0.0.0:8080")
	_ = dl.SetDefault(InstanceID, "")

	// Security
	_ = dl.SetDefault(TlsKeyFile, "")
	_ = dl.SetDefault(TlsCertFile, "")
	_ = dl.SetDefault(ConfJobAddress, "nats://localhost:4222")

	// Database (MongoDB)
	_ = dl.SetDefault(ConfMongoTls, true)
	_ = dl.SetDefault(MongoDSN, "localhost:27017")

	// Cache (Redis)
	_ = dl.SetDefault(RedisDSN, "localhost:6379")

	// Debugging
	_ = dl.SetDefault(LogLevel, 2)

	// ADP Configs
	_ = dl.SetDefault(ADPUsername, "ronak")
	_ = dl.SetDefault(ADPPassword, "E2e2374k19743")
	_ = dl.SetDefault(ADPMessageUrl, "https://ws.adpdigital.com/url/send")

	// SMTP
	_ = dl.SetDefault(SmtpHost, "mta")
	_ = dl.SetDefault(SmtpPort, 25)
	_ = dl.SetDefault(SmtpUser, "user")
	_ = dl.SetDefault(SmtpPass, "pa$$word")

	// Extra Configs
	_ = dl.SetDefault(MonitorAccessToken, "!@NES##monitor##TED@!")
	_ = dl.SetDefault(FileSystemKey, "testKey")

	_Onion = onion.New()
	_ = _Onion.AddLayer(dl)
	_Onion.AddLazyLayer(extraenv.NewExtraEnvLayer("NST"))
}

func GetString(key string) string {
	return _Onion.GetString(key)
}

func GetInt(key string) int {
	return _Onion.GetInt(key)
}

func GetInt64(key string) int64 {
	return _Onion.GetInt64(key)
}

func GetBool(key string) bool {
	return _Onion.GetBool(key)
}
