package job

import (
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
	ticker := time.NewTicker(10 * time.Second)

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
	// Clean up the backup files
	c.clean1minInterval()
	c.clean10minInterval()
}

// get all the files updated in last 15 min and space then in 1 min interval
func (c *BackupCleaner) clean1minInterval() {
	// Get all the files updated in last 10 min
	backups, err := c.store.GetDocumentByUpdatedTime(time.Now().Add(-15 * time.Minute))
	if err != nil {
		logrus.Error("Error getting the files updated in last 15 min: ", err)
		return
	}

	// Clean up the backup files
	for _, backup := range backups {
		logrus.Infof("Cleaning up the backup file: %s", backup)
	}
}

// get all the files updated in last 20min and space then in 10 min interval
func (c *BackupCleaner) clean10minInterval() {}
