package api

import (
    "time"
    "git.ronaksoftware.com/nested/server/model"
    "git.ronaksoftware.com/nested/server/server-ntfy/client"
    "log"
    "sync"
    "gopkg.in/fzerorubigd/onion.v3"
)

// BackgroundJob are runnable structures handler background workers
type BackgroundJob struct {
    chShutdown chan bool
    server     *API
    period     time.Duration
    job        func(worker *BackgroundJob)
}

func NewBackgroundJob(server *API, d time.Duration, f func(w *BackgroundJob)) *BackgroundJob {
    w := new(BackgroundJob)
    w.period = d
    w.job = f
    w.server = server
    w.chShutdown = make(chan bool)
    return w
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
        nested.SYS_INFO_USERAPI,
        bundleID,
        nested.SystemInfo(),
    )
    log.Println("Report Debouncer Called.")
}

// JobOverdueTasks
func JobOverdueTasks(b *BackgroundJob) {
    var ntfyClient *ntfy.Client
    for ntfyClient == nil {
        ntfyClient = ntfy.NewClient(b.Config().GetString("JOB_ADDRESS"), b.Model())
        if ntfyClient == nil {
            log.Println("BackgroundJob::TaskMonitor::Failed to connected to ntfy server")
        }
        time.Sleep(5 * time.Second)
    }
    ntfyClient.SetDomain(b.Config().GetString("DOMAIN"))
    defer ntfyClient.Close()

    buckets := b.Model().TimeBucket.GetBucketsBefore(nested.Timestamp())
    for _, bucket := range buckets {
        overdueTasks := b.Model().Task.GetTasksByIDs(bucket.OverdueTasks)
        for i, task := range overdueTasks {
            // Set task's status to overdue
            switch task.Status {
            case nested.TASK_STATUS_COMPLETED, nested.TASK_STATUS_FAILED, nested.TASK_STATUS_HOLD, nested.TASK_STATUS_CANCELED:
				b.Model().TimeBucket.Remove(bucket.ID)
				continue
            default:
                task.UpdateStatus("nested", nested.TASK_STATUS_OVERDUE)
            }

            // Notify Assignor of the task
            if len(task.AssigneeID) > 0 {
                n1 := b.Model().Notification.TaskOverdue(task.AssignorID, &overdueTasks[i])
                ntfyClient.ExternalPushNotification(n1)
                ntfyClient.InternalNotificationSyncPush([]string{task.AssigneeID}, nested.NOTIFICATION_TYPE_TASK_OVER_DUE)
            }
            // Notify Assignee of the task
            if len(task.AssignorID) > 0 && task.AssignorID != task.AssigneeID {
                n2 := b.Model().Notification.TaskOverdue(task.AssigneeID, &overdueTasks[i])
                ntfyClient.ExternalPushNotification(n2)
                ntfyClient.InternalNotificationSyncPush([]string{task.AssignorID}, nested.NOTIFICATION_TYPE_TASK_OVER_DUE)
            }


        }
        b.Model().TimeBucket.Remove(bucket.ID)
    }

}

// JobLicenseManager
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
