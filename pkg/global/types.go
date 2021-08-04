package global

/*
   Creation Time: 2021 - Aug - 04
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type PostAction int

const (
	PostActivityActionCommentAdd    PostAction = 0x002
	PostActivityActionCommentRemove PostAction = 0x003
	PostActivityActionLabelAdd      PostAction = 0x011
	PostActivityActionLabelRemove   PostAction = 0x012
	PostActivityActionEdited        PostAction = 0x015
	PostActivityActionPlaceMove     PostAction = 0x016
	PostActivityActionPlaceAttach   PostAction = 0x017
)

type TaskAction int

const (
	TaskActivityWatcherAdded      TaskAction = 0x0001
	TaskActivityWatcherRemoved    TaskAction = 0x0002
	TaskActivityAttachmentAdded   TaskAction = 0x0003
	TaskActivityAttachmentRemoved TaskAction = 0x0004
	TaskActivityComment           TaskAction = 0x0006
	TaskActivityTitleChanged      TaskAction = 0x0007
	TaskActivityDescChanged       TaskAction = 0x0008
	TaskActivityCandidateAdded    TaskAction = 0x0011
	TaskActivityCandidateRemoved  TaskAction = 0x0012
	TaskActivityTodoAdded         TaskAction = 0x0013
	TaskActivityTodoRemoved       TaskAction = 0x0014
	TaskActivityTodoChanged       TaskAction = 0x0015
	TaskActivityTodoDone          TaskAction = 0x0016
	TaskActivityTodoUndone        TaskAction = 0x0017
	TaskActivityStatusChanged     TaskAction = 0x0018
	TaskActivityLabelAdded        TaskAction = 0x0019
	TaskActivityLabelRemoved      TaskAction = 0x0020
	TaskActivityDueDateUpdated    TaskAction = 0x0021
	TaskActivityDueDateRemoved    TaskAction = 0x0022
	TaskActivityCreated           TaskAction = 0x0023
	TaskActivityAssigneeChanged   TaskAction = 0x0024
	TaskActivityEditorAdded       TaskAction = 0x0025
	TaskActivityEditorRemoved     TaskAction = 0x0026
	TaskActivityUpdated           TaskAction = 0x0100
)
