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
	CacheLifetime         = 3600 // Seconds
	RegisterModeAdminOnly = REGISTER_MODE_ADMIN_ONLY
	DefaultMaxUploadSize  = "100MB"

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
	COLLECTION_SYSTEM_INTERNAL          = "model.internal"
	COLLECTION_APPS                     = "apps"
	COLLECTION_ACCOUNTS                 = "accounts"
	COLLECTION_ACCOUNTS_DATA            = "accounts.data" // Account's clients data
	COLLECTION_ACCOUNTS_DEVICES         = "accounts.devices"
	COLLECTION_ACCOUNTS_TRUSTED         = "accounts.trusted"
	COLLECTION_ACCOUNTS_RECIPIENTS      = "accounts.recipients" // Account's most related emails
	COLLECTION_ACCOUNTS_PLACES          = "accounts.places"     // Account's most related places
	COLLECTION_ACCOUNTS_ACCOUNTS        = "accounts.accounts"   // Account's most related accounts
	COLLECTION_ACCOUNTS_POSTS           = "accounts.posts"      // Account's bookmarked posts
	COLLECTION_ACCOUNTS_LABELS          = "accounts.labels"
	COLLECTION_ACCOUNTS_SEARCH_HISTORY  = "accounts.search.history"
	COLLECTION_CONTACTS                 = "contacts"
	COLLECTION_FILES                    = "files"
	COLLECTION_HOOKS                    = "hooks"
	COLLECTION_NOTIFICATIONS            = "notifications"
	COLLECTION_LABELS                   = "labels"
	COLLECTION_LABELS_REQUESTS          = "labels.requests"
	COLLECTION_PHONES                   = "phones"
	COLLECTION_PLACES                   = "places"
	COLLECTION_PLACES_ACTIVITIES        = "places.activities"
	COLLECTION_PLACES_DEFAULT           = "places.default"
	COLLECTION_PLACES_GROUPS            = "places.groups"
	COLLECTION_PLACES_BLOCKED_ADDRESSES = "places.blocked_addresses"
	COLLECTION_POSTS                    = "posts"
	COLLECTION_POSTS_ACTIVITIES         = "posts.activities"
	COLLECTION_POSTS_COMMENTS           = "posts.comments"
	COLLECTION_POSTS_READS              = "posts.reads"
	COLLECTION_POSTS_READS_COUNTERS     = "posts.reads.counters"
	COLLECTION_POSTS_READS_ACCOUNTS     = "posts.reads.accounts"
	COLLECTION_POSTS_WATCHERS           = "posts.watchers"
	COLLECTION_POSTS_FILES              = "posts.files"
	COLLECTION_REPORTS_COUNTERS         = "reports.counters"
	COLLECTION_SESSIONS                 = "sessions"
	COLLECTION_SYS_RESERVED_WORDS       = "nsys.reserved_words"
	COLLECTION_SEARCH_INDEX_PLACES      = "search.index.place"
	COLLECTION_TASKS                    = "tasks"
	COLLECTION_TASKS_ACTIVITIES         = "tasks.activities"
	COLLECTION_TASKS_FILES              = "tasks.files"
	COLLECTION_TOKENS_APPS              = "tokens.apps"
	COLLECTION_TOKENS_FILES             = "tokens.files"
	COLLECTION_TOKENS_LOGINS            = "tokens.logins"
	COLLECTION_VERIFICATIONS            = "verifications"
	COLLECTION_LOGS                     = "logs"
	COLLECTION_TIME_BUCKETS             = "time_buckets"
)

// ErrorCode
type ErrorCode int

const (
	ERR_UNKNOWN     ErrorCode = 0x00
	ERR_ACCESS      ErrorCode = 0x01
	ERR_UNAVAILABLE ErrorCode = 0x02
	ERR_INVALID     ErrorCode = 0x03
	ERR_INCOMPLETE  ErrorCode = 0x04
	ERR_DUPLICATE   ErrorCode = 0x05
	ERR_LIMIT       ErrorCode = 0x06
	ERR_TIMEOUT     ErrorCode = 0x07
	ERR_SESSION     ErrorCode = 0x08
)

// REGISTER MODE
const (
	REGISTER_MODE_EVERYONE   int = 0x01
	REGISTER_MODE_ADMIN_ONLY int = 0x02
)

// DEBUG LEVELS
const (
	DEBUG_LEVEL_0 int = 0
	DEBUG_LEVEL_1 int = 1
	DEBUG_LEVEL_2 int = 2
)

// SYSTEM COUNTERS
const (
	SYSTEM_COUNTERS_ENABLED_ACCOUNTS  = "enabled_accounts"
	SYSTEM_COUNTERS_DISABLED_ACCOUNTS = "disabled_accounts"
	SYSTEM_COUNTERS_PERSONAL_PLACES   = "personal_places"
	SYSTEM_COUNTERS_GRAND_PLACES      = "grand_places"
	SYSTEM_COUNTERS_LOCKED_PLACES     = "locked_places"
	SYSTEM_COUNTERS_UNLOCKED_PLACES   = "unlocked_places"
)

// SYSTEM CONSTANTS
const (
	SYSTEM_CONSTANTS_MODEL_VERSION            = "model_version"
	SYSTEM_CONSTANTS_CACHE_LIFETIME           = "cache_lifetime"
	SYSTEM_CONSTANTS_POST_MAX_TARGETS         = "post_max_targets"
	SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS     = "post_max_attachments"
	SYSTEM_CONSTANTS_POST_RETRACT_TIME        = "post_retract_time"
	SYSTEM_CONSTANTS_POST_MAX_LABELS          = "post_max_labels"
	SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT = "account_grandplaces_limit"
	SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN       = "place_max_children"
	SYSTEM_CONSTANTS_PLACE_MAX_CREATORS       = "place_max_creators"
	SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS     = "place_max_keyholders"
	SYSTEM_CONSTANTS_PLACE_MAX_LEVEL          = "place_max_level"
	SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS        = "label_max_members"
	SYSTEM_CONSTANTS_REGISTER_MODE            = "register_mode"
	SYSTEM_CONSTANTS_UPLOAD_MAX_SIZE          = "upload_max_size"
	SYSTEM_CONSTANTS_COMPANY_NAME             = "company_name"
	SYSTEM_CONSTANTS_COMPANY_DESC             = "company_desc"
	SYSTEM_CONSTANTS_COMPANY_LOGO             = "company_logo"
	SYSTEM_CONSTANTS_SYSTEM_LANG              = "system_lang"
	SYSTEM_CONSTANTS_MAGIC_NUMBER             = "magic_number"
	SYSTEM_CONSTANTS_LICENSE_KEY              = "license_key"

	SYSTEM_CONSTANTS_CACHE_LIFETIME_UL           int = 86400 // seconds
	SYSTEM_CONSTANTS_CACHE_LIFETIME_LL           int = 60
	SYSTEM_CONSTANTS_POST_MAX_TARGETS_UL         int = 50
	SYSTEM_CONSTANTS_POST_MAX_TARGETS_LL         int = 5
	SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS_LL     int = 5
	SYSTEM_CONSTANTS_POST_MAX_ATTACHMENTS_UL     int = 50
	SYSTEM_CONSTANTS_POST_MAX_LABELS_LL          int = 1
	SYSTEM_CONSTANTS_POST_MAX_LABELS_UL          int = 25
	SYSTEM_CONSTANTS_POST_RETRACT_TIME_LL        int = 0
	SYSTEM_CONSTANTS_POST_RETRACT_TIME_UL        int = 86400000 // milliseconds
	SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_LL int = 0
	SYSTEM_CONSTANTS_ACCOUNT_GRANDPLACE_LIMIT_UL int = 1000
	SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_LL       int = 0
	SYSTEM_CONSTANTS_PLACE_MAX_CHILDREN_UL       int = 50
	SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_LL       int = 1
	SYSTEM_CONSTANTS_PLACE_MAX_CREATORS_UL       int = 200
	SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_LL     int = 1
	SYSTEM_CONSTANTS_PLACE_MAX_KEYHOLDERS_UL     int = 2500
	SYSTEM_CONSTANTS_PLACE_MAX_LEVEL_LL          int = 3
	SYSTEM_CONSTANTS_PLACE_MAX_LEVEL_UL          int = 5
	SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS_UL        int = 50
	SYSTEM_CONSTANTS_LABEL_MAX_MEMBERS_LL        int = 1
)
