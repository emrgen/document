package job

import (
	"context"
	goset "github.com/deckarep/golang-set/v2"
	"github.com/emrgen/document/internal/store"
	"github.com/sirupsen/logrus"
	"time"
)

// BackupCleaner is a job that cleans up the backup files.
type BackupCleaner struct {
	store store.Store
	done  chan struct{}
}

// NewBackupCleaner creates a new BackupCleaner instance.
func NewBackupCleaner(store store.Store) *BackupCleaner {
	return &BackupCleaner{
		store: store,
		done:  make(chan struct{}),
	}
}

func (c *BackupCleaner) Stop() {
	close(c.done)
}

func (c *BackupCleaner) Run() {
	ticker := time.NewTicker(4 * time.Second)

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			c.clean()
		}
	}
}

func (c *BackupCleaner) clean() {
	c.window(10 * time.Minute)
	//c.window(30 * time.Minute)
}

// get all the files updated in last 15 min and space then in 1 min interval
func (c *BackupCleaner) window(duration time.Duration) {
	logrus.Infof("Cleaning up the backup files: %s", time.Now().Format(time.RFC3339))

	// Get all the files updated in last 10 min
	backups, err := c.store.GetDocumentByUpdatedTime(time.Now().Add(-2*duration), time.Now())
	if err != nil {
		logrus.Error("Error getting the files updated in last 15 min: ", err)
		return
	}

	remove := make(map[string]goset.Set[int64])
	lastBackupTime := time.Time{}
	logrus.Infof("isZero: %v", lastBackupTime.IsZero())
	for _, backup := range backups {
		logrus.Infof("version: %v", backup.Version)
		if lastBackupTime.IsZero() {
			lastBackupTime = backup.CreatedAt.Round(duration)
			continue
		}

		backupTime := backup.CreatedAt.Round(duration)
		if backupTime.Equal(lastBackupTime) {
			if _, ok := remove[backup.ID]; !ok {
				remove[backup.ID] = goset.NewSet[int64]()
			}

			remove[backup.ID].Add(backup.Version)
		} else {
			lastBackupTime = backupTime
		}
	}

	err = c.store.DeleteDocumentBackups(context.TODO(), remove)
	if err != nil {
		logrus.Error("Error deleting the backups: ", err)
		return
	}

	logrus.Infof("Removing %v backups", remove)
}

// get all the files updated in last 20min and space then in 10 min interval
func (c *BackupCleaner) clean10minInterval() {}
