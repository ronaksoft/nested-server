package nested

import (
	"git.ronaksoft.com/nested/server/pkg/log"
	"go.uber.org/zap"
	"time"
)

/*
   Creation Time: 2018 - May - 13
   Created by:  (ehsan)
   Maintainers:
       1.  (ehsan)
   Auditor: Ehsan N. Moosa
   Copyright Ronak Software Group 2018
*/

const (
	CheckupSyncPlaceCounters              = "SYNC_PLACE_COUNTERS"
	CheckupSyncPostCounters               = "SYNC_POST_COUNTERS"
	CheckupSyncTaskCounters               = "SYNC_TASK_COUNTERS"
	CheckupSyncLabelCounters              = "SYNC_LABEL_COUNTERS"
	CheckupSyncFileRefCounters            = "SYNC_FILE_REF_COUNTERS"
	CheckupSyncSystemCounters             = "SYNC_SYSTEM_COUNTERS"
	CheckupCleanupTasks                   = "CLEANUP_TASKS"
	CheckupCleanupSessions                = "CLEANUP_SESSIONS"
	CheckupCleanupPosts                   = "CLEANUP_POSTS"
	CheckupCleanupTempFiles               = "CLEANUP_TEMP_FILES"
	CheckupFixReferredTempFiles           = "FIX_REFERRED_TEMP_FILES"
	CheckupFixCollectionSearchIndexPlaces = "FIX_COLLECTION_SEARCH_INDEX_PLACES"
	CheckupAddContentToPost               = "CHECKUP_ADD_CONTENT_TO_POST"
)

var (
	_CheckupRoutines = map[string]func(){
		CheckupSyncPlaceCounters:              SyncPlaceCounters,
		CheckupSyncLabelCounters:              SyncLabelCounters,
		CheckupSyncPostCounters:               SyncPostCounters,
		CheckupSyncTaskCounters:               SyncTaskCounters,
		CheckupSyncFileRefCounters:            SyncFileRefCounters,
		CheckupSyncSystemCounters:             SyncSystemCounters,
		CheckupCleanupPosts:                   CleanupPosts,
		CheckupCleanupSessions:                CleanupSessions,
		CheckupCleanupTasks:                   CleanupTasks,
		CheckupAddContentToPost:               AddContentToPost,
		CheckupFixReferredTempFiles:           FixReferredTmpFiles,
		CheckupFixCollectionSearchIndexPlaces: FixSearchIndexPlacesCollection,
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
		for r := range _CheckupRoutines {
			routines = append(routines, r)
		}
	}

	for _, key := range routines {
		if checkupRoutine, ok := _CheckupRoutines[key]; ok {
			_CurrentRunningCheckup = key
			_FinishedRoutines = append(_FinishedRoutines, _CurrentRunningCheckup)
			log.Info("Checkup routine started", zap.String("Name", key))
			startTime := time.Now()
			checkupRoutine()
			log.Info("Checkup routine finished", zap.String("Name", key), zap.Duration("D", time.Now().Sub(startTime)))
		}
	}
	return true
}

func toggleRunState() {
	_Running = !_Running
}
