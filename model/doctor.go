package nested

/*
   Creation Time: 2018 - May - 13
   Created by:  (ehsan)
   Maintainers:
       1.  (ehsan)
   Auditor: Ehsan N. Moosa
   Copyright Ronak Software Group 2018
*/

const (
	CHECKUP_SYNC_PLACE_COUNTERS                = "SYNC_PLACE_COUNTERS"
	CHECKUP_SYNC_POST_COUNTERS                 = "SYNC_POST_COUNTERS"
	CHECKUP_SYNC_TASK_COUNTERS                 = "SYNC_TASK_COUNTERS"
	CHECKUP_SYNC_LABEL_COUNTERS                = "SYNC_LABEL_COUNTERS"
	CHECKUP_SYNC_FILE_REF_COUNTERS             = "SYNC_FILE_REF_COUNTERS"
	CHECKUP_SYNC_SYSTEM_COUNTERS               = "SYNC_SYSTEM_COUNTERS"
	CHECKUP_CLEANUP_TASKS                      = "CLEANUP_TASKS"
	CHECKUP_CLEANUP_SESSIONS                   = "CLEANUP_SESSIONS"
	CHECKUP_CLEANUP_POSTS                      = "CLEANUP_POSTS"
	CHECKUP_CLEANUP_TEMP_FILES                 = "CLEANUP_TEMP_FILES"
	CHECKUP_FIX_REFERRED_TEMP_FILES            = "FIX_REFERRED_TEMP_FILES"
	CHECKUP_FIX_COLLECTION_SEARCH_INDEX_PLACES = "FIX_COLLECTION_SEARCH_INDEX_PLACES"
	CHECKUP_ADD_CONTENT_TO_POST                = "CHECKUP_ADD_CONTENT_TO_POST"
)

var (
	_CheckupRoutines = map[string]func(){
		CHECKUP_SYNC_PLACE_COUNTERS:    SyncPlaceCounters,
		CHECKUP_SYNC_LABEL_COUNTERS:    SyncLabelCounters,
		CHECKUP_SYNC_POST_COUNTERS:     SyncPostCounters,
		CHECKUP_SYNC_TASK_COUNTERS:     SyncTaskCounters,
		CHECKUP_SYNC_FILE_REF_COUNTERS: SyncFileRefCounters,
		CHECKUP_SYNC_SYSTEM_COUNTERS:   SyncSystemCounters,
		CHECKUP_CLEANUP_POSTS:          CleanupPosts,
		CHECKUP_CLEANUP_SESSIONS:       CleanupSessions,
		CHECKUP_CLEANUP_TASKS:          CleanupTasks,
		CHECKUP_ADD_CONTENT_TO_POST:     AddContentToPost,
		CHECKUP_FIX_REFERRED_TEMP_FILES:            FixReferredTmpFiles,
		CHECKUP_FIX_COLLECTION_SEARCH_INDEX_PLACES: FixSearchIndexPlacesCollection,
	}
	_FinishedRoutines      []string
	_CurrentRunningCheckup string
	_Running               bool
)

func RunDoctor(routines []string) bool {
	if _Running {
		return false
	}
	toggleRunState()
	defer toggleRunState()

	_FinishedRoutines = make([]string, 0, len(_CheckupRoutines))
	if routines == nil {
		for key, checkupRoutine := range _CheckupRoutines {
			_CurrentRunningCheckup = key
			_FinishedRoutines = append(_FinishedRoutines, _CurrentRunningCheckup)
			checkupRoutine()
		}
	} else {
		for _, key := range routines {
			if checkupRoutine, ok := _CheckupRoutines[key]; ok {
				_CurrentRunningCheckup = key
				_FinishedRoutines = append(_FinishedRoutines, _CurrentRunningCheckup)
				checkupRoutine()
			}
		}
	}
	return true
}

func toggleRunState() {
	_Running = !_Running
}
