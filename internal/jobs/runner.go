package jobs

import (
	"sync"

	mapset "github.com/deckarep/golang-set/v2"
	cron "github.com/robfig/cron"
	"github.com/sirupsen/logrus"
)

type Job interface {
	Run()
}

type CronJob interface {
	Schedule() string
	Job
}

type TaskExecutor struct {
	cron            *cron.Cron
	jobs            []Job
	cronJobs        []CronJob
	runningJobs     mapset.Set[Job]
	runningCronJobs mapset.Set[CronJob]
	muJobs          sync.Mutex
	muCronJobs      sync.Mutex
}

func NewTaskExecutor(jobs []Job, cronJobs []CronJob) *TaskExecutor {
	return &TaskExecutor{
		cron:            cron.New(),
		jobs:            jobs,
		cronJobs:        cronJobs,
		runningCronJobs: mapset.NewSet[CronJob](),
		runningJobs:     mapset.NewSet[Job](),
		muJobs:          sync.Mutex{},
		muCronJobs:      sync.Mutex{},
	}
}

// Run the jobs in its own goroutine inside the cron.
func (t *TaskExecutor) Run() {
	for _, job := range t.cronJobs {
		err := t.cron.AddFunc(job.Schedule(), func() {
			t.muCronJobs.Lock()

			if t.runningCronJobs.Contains(job) {
				logrus.Warn("task is already scheduled")
				return
			}

			t.runningCronJobs.Add(job)
			t.muCronJobs.Unlock()

			defer func() {
				t.muCronJobs.Lock()
				defer t.muCronJobs.Unlock()
				t.runningCronJobs.Remove(job)
			}()

			job.Run()
		})

		if err != nil {
			logrus.Errorf("failed to add task to cron: %v", err)
			panic(err)
		}
	}

	for _, job := range t.jobs {
		t.cron.AddFunc("@every 1s", func() {
			t.muJobs.Lock()

			if t.runningJobs.Contains(job) {
				logrus.Warn("task is already running")
				return
			}

			t.runningJobs.Add(job)
			t.muJobs.Unlock()

			defer func() {
				t.muJobs.Lock()
				defer t.muJobs.Unlock()
				t.runningJobs.Remove(job)
			}()

			job.Run()
		})
	}

	t.cron.Start()
}

func (t *TaskExecutor) Stop() {
	logrus.Infof("stopping all tasks")
	t.cron.Stop()
}
