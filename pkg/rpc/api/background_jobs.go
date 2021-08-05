package api

import (
	"sync"
	"time"

	"git.ronaksoft.com/nested/server/nested"
	"gopkg.in/fzerorubigd/onion.v3"
)

// BackgroundJob are runnable structures handler background workers
type BackgroundJob struct {
	chShutdown chan bool
	server     *Server
	period     time.Duration
	job        func(worker *BackgroundJob)
}

func NewBackgroundJob(server *Server, d time.Duration, f func(w *BackgroundJob)) *BackgroundJob {
	bg := &BackgroundJob{
		period:     d,
		job:        f,
		server:     server,
		chShutdown: make(chan bool),
	}

	return bg
}

func (bw *BackgroundJob) Run(wGroup *sync.WaitGroup) {
	wGroup.Add(1)
	defer wGroup.Done()
	for {
		select {
		case <-time.After(bw.period):
			bw.job(bw)
		case <-bw.chShutdown:
			return
		}
	}
}

func (bw *BackgroundJob) Shutdown() {
	bw.chShutdown <- true
}

func (bw *BackgroundJob) Config() *onion.Onion {
	return bw.server.config
}

func (bw *BackgroundJob) Model() *nested.Manager {
	return bw.server.model
}

// JobReporter
// Report De-bouncer Routine
func JobReporter(b *BackgroundJob) {
	bundleID := b.Config().GetString("BUNDLE_ID")

	// Flush counters into DB
	b.Model().Report.FlushToDB()

	b.Model().System.SetSystemInfo(
		nested.SysInfoUserAPI,
		bundleID,
		nested.SystemInfo(),
	)
}

func JobOverdueTasks(b *BackgroundJob) {

	buckets := b.Model().TimeBucket.GetBucketsBefore(nested.Timestamp())
	for _, bucket := range buckets {
		overdueTasks := b.Model().Task.GetTasksByIDs(bucket.OverdueTasks)
		for i, task := range overdueTasks {
			// Set task's status to overdue
			switch task.Status {
			case nested.TaskStatusCompleted, nested.TaskStatusFailed, nested.TaskStatusHold, nested.TaskStatusCanceled:
				b.Model().TimeBucket.Remove(bucket.ID)
				continue
			default:
				task.UpdateStatus("nested", nested.TaskStatusOverdue)
			}

			// Notify Assignor of the task
			if len(task.AssigneeID) > 0 {
				n1 := b.Model().Notification.TaskOverdue(task.AssignorID, &overdueTasks[i])
				b.server.pusher.ExternalPushNotification(n1)
				b.server.pusher.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NotificationTypeTaskOverDue)
			}
			// Notify Assignee of the task
			if len(task.AssignorID) > 0 && task.AssignorID != task.AssigneeID {
				n2 := b.Model().Notification.TaskOverdue(task.AssigneeID, &overdueTasks[i])
				b.server.pusher.ExternalPushNotification(n2)
				b.server.pusher.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NotificationTypeTaskOverDue)
			}

		}
		b.Model().TimeBucket.Remove(bucket.ID)
	}

}

func JobLicenseManager(b *BackgroundJob) {
	license := b.Model().License.Get()
	licenseTime := time.Unix(int64(license.ExpireDate/1000), 0)
	hours := time.Now().Sub(licenseTime).Hours()

	if b.Model().License.IsExpired() {
		b.server.flags.LicenseExpired = true
		if hours > 1440 { // More than 2 Months
			b.server.flags.LicenseSlowMode = 2
		} else if hours > 720 { // More than 1 Months
			b.server.flags.LicenseSlowMode = 1
		} else {
			b.server.flags.LicenseSlowMode = 0
		}
	} else {
		b.server.flags.LicenseExpired = false
	}
}
