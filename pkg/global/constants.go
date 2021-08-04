package global

// Unchangeable Default Parameters
const (
	DefaultMaxVerificationAttempts = 3
	DefaultMaxResultLimit          = 100
	DefaultMaxBookmarkedPlaces     = 1000
	DefaultMaxClientObjCount       = 10
	DefaultMaxClientObjSize        = 102400
	DefaultMaxPlaceName            = 64
	DefaultMaxAccountName          = 32
	DefaultMaxPlaceID              = 128
	DefaultMaxLabelTitle           = 32
	DefaultModelVersion            = 17

	DefaultRegexPlaceID      = "^[a-zA-Z][a-zA-Z0-9-_]{0,30}[a-zA-Z0-9]$"
	DefaultRegexGrandPlaceID = "^[a-zA-Z][a-zA-Z0-9-_]{1,30}[a-zA-Z0-9]$"
	DefaultRegexAccountID    = "^[a-zA-Z][a-zA-Z0-9-_]{1,30}[a-zA-Z0-9]$"
	DefaultRegexEmail        = "^[a-z0-9._%+\\-]+@[a-z0-9.\\-]+\\.[a-z]{2,4}$"
)

// Minimum Client Versions
const (
	AndroidCurrentSdkVersion = 339
	AndroidMinSdkVersion     = 338
	IosCurrentSdkVersion     = 10
	IosMinSdkVersion         = 10
)

// Supported Client Platforms
const (
	PlatformAndroid = "android"
	PlatformIOS     = "ios"
	PlatformFirefox = "firefox"
	PlatformChrome  = "chrome"
	PlatformSafari  = "safari"
)

// Adjustable Default Parameters
var (
	CacheLifetime        = 3600 // Seconds
	RegisterMode         = RegisterModeAdminOnly
	DefaultMaxUploadSize = "100MB"

	DefaultPlaceMaxChildren   = 10
	DefaultPlaceMaxCreators   = 5
	DefaultPlaceMaxKeyHolders = 25
	DefaultPlaceMaxLevel      = 3

	DefaultPostMaxAttachments        = 50
	DefaultPostMaxTargets            = 20
	DefaultPostMaxLabels             = 10
	DefaultPostRetractTime    uint64 = 86400000 // 24h

	DefaultAccountGrandPlaces = 2

	DefaultLabelMaxMembers = 50

	DefaultCompanyName = "Nested"
	DefaultCompanyDesc = "Team Communication Platform"
	DefaultCompanyLogo = ""
	DefaultMagicNumber = "989121228718"
	DefaultSystemLang  = "en"

	DbName    string
	StoreName string
)

// MONGODB COLLECTIONS
const (
	CollectionSystemInternal         = "model.internal"
	CollectionApps                   = "apps"
	CollectionAccounts               = "accounts"
	CollectionAccountsData           = "accounts.data" // Account's clients data
	CollectionAccountsDevices        = "accounts.devices"
	CollectionAccountsTrusted        = "accounts.trusted"
	CollectionAccountsRecipients     = "accounts.recipients" // Account's most related emails
	CollectionAccountsPlaces         = "accounts.places"     // Account's most related places
	CollectionAccountsAccounts       = "accounts.accounts"   // Account's most related accounts
	CollectionAccountsPosts          = "accounts.posts"      // Account's bookmarked posts
	CollectionAccountsLabels         = "accounts.labels"
	CollectionAccountsSearchHistory  = "accounts.search.history"
	CollectionContacts               = "contacts"
	CollectionFiles                  = "files"
	CollectionHooks                  = "hooks"
	CollectionNotifications          = "notifications"
	CollectionLabels                 = "labels"
	CollectionLabelsRequests         = "labels.requests"
	CollectionPhones                 = "phones"
	CollectionPlaces                 = "places"
	CollectionPlacesActivities       = "places.activities"
	CollectionPlacesDefault          = "places.default"
	CollectionPlacesGroups           = "places.groups"
	CollectionPlacesBlockedAddresses = "places.blocked_addresses"
	CollectionPosts                  = "posts"
	CollectionPostsActivities        = "posts.activities"
	CollectionPostsComments          = "posts.comments"
	CollectionPostsReads             = "posts.reads"
	CollectionPostsReadsCounters     = "posts.reads.counters"
	CollectionPostsReadsAccounts     = "posts.reads.accounts"
	CollectionPostsWatchers          = "posts.watchers"
	CollectionPostsFiles             = "posts.files"
	CollectionReportsCounters        = "reports.counters"
	CollectionSessions               = "sessions"
	CollectionSysReservedWords       = "nsys.reserved_words"
	CollectionSearchIndexPlaces      = "search.index.place"
	CollectionTasks                  = "tasks"
	CollectionTasksActivities        = "tasks.activities"
	CollectionTasksFiles             = "tasks.files"
	CollectionTokensApps             = "tokens.apps"
	CollectionTokensFiles            = "tokens.files"
	CollectionTokensLogins           = "tokens.logins"
	CollectionVerifications          = "verifications"
	CollectionLogs                   = "logs"
	CollectionTimeBuckets            = "time_buckets"
)

type ErrorCode int

const (
	ErrUnknown     ErrorCode = 0x00
	ErrAccess      ErrorCode = 0x01
	ErrUnavailable ErrorCode = 0x02
	ErrInvalid     ErrorCode = 0x03
	ErrIncomplete  ErrorCode = 0x04
	ErrDuplicate   ErrorCode = 0x05
	ErrLimit       ErrorCode = 0x06
	ErrTimeout     ErrorCode = 0x07
	ErrSession     ErrorCode = 0x08
)

// REGISTER MODE
const (
	RegisterModeEveryone  int = 0x01
	RegisterModeAdminOnly int = 0x02
)

// SYSTEM COUNTERS
const (
	SystemCountersEnabledAccounts  = "enabled_accounts"
	SystemCountersDisabledAccounts = "disabled_accounts"
	SystemCountersPersonalPlaces   = "personal_places"
	SystemCountersGrandPlaces      = "grand_places"
	SystemCountersLockedPlaces     = "locked_places"
	SystemCountersUnlockedPlaces   = "unlocked_places"
)

// SYSTEM CONSTANTS
const (
	SystemConstantsModelVersion           = "model_version"
	SystemConstantsCacheLifetime          = "cache_lifetime"
	SystemConstantsPostMaxTargets         = "post_max_targets"
	SystemConstantsPostMaxAttachments     = "post_max_attachments"
	SystemConstantsPostRetractTime        = "post_retract_time"
	SystemConstantsPostMaxLabels          = "post_max_labels"
	SystemConstantsAccountGrandPlaceLimit = "account_grandplaces_limit"
	SystemConstantsPlaceMaxChildren       = "place_max_children"
	SystemConstantsPlaceMaxCreators       = "place_max_creators"
	SystemConstantsPlaceMaxKeyHolders     = "place_max_keyholders"
	SystemConstantsPlaceMaxLevel          = "place_max_level"
	SystemConstantsLabelMaxMembers        = "label_max_members"
	SystemConstantsRegisterMode           = "register_mode"
	SystemConstantsUploadMaxSize          = "upload_max_size"
	SystemConstantsCompanyName            = "company_name"
	SystemConstantsCompanyDesc            = "company_desc"
	SystemConstantsCompanyLogo            = "company_logo"
	SystemConstantsSystemLang             = "system_lang"
	SystemConstantsMagicNumber            = "magic_number"
	SystemConstantsLicenseKey             = "license_key"

	SystemConstantsCacheLifetimeUL          int = 86400 // seconds
	SystemConstantsCacheLifetimeLL          int = 60
	SystemConstantsPostMaxTargetsUL         int = 50
	SystemConstantsPostMaxTargetsLL         int = 5
	SystemConstantsPostMaxAttachmentsLL     int = 5
	SystemConstantsPostMaxAttachmentsUL     int = 50
	SystemConstantsPostMaxLabelsLL          int = 1
	SystemConstantsPostMaxLabelsUL          int = 25
	SystemConstantsPostRetractTimeLL        int = 0
	SystemConstantsPostRetractTimeUL        int = 86400000 // milliseconds
	SystemConstantsAccountGrandPlaceLimitLL int = 0
	SystemConstantsAccountGrandPlaceLimitUL int = 1000
	SystemConstantsPlaceMaxChildrenLL       int = 0
	SystemConstantsPlaceMaxChildrenUL       int = 50
	SystemConstantsPlaceMaxCreatorsLL       int = 1
	SystemConstantsPlaceMaxCreatorsUl       int = 200
	SystemConstantsPlaceMaxKeyHoldersLL     int = 1
	SystemConstantsPlaceMaxKeyHoldersUL     int = 2500
	SystemConstantsPlaceMaxLevelLL          int = 3
	SystemConstantsPlaceMaxLevelUL          int = 5
	SystemConstantsLabelMaxMembersUL        int = 50
	SystemConstantsLabelMaxMembersLL        int = 1
)
