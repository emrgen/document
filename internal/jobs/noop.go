package jobs

import "github.com/sirupsen/logrus"

type LoggerTask struct {
}

func NewLoggerTask() *LoggerTask {
	return &LoggerTask{}
}

func (l *LoggerTask) ID() string {
	return "logger"
}

func (l *LoggerTask) Name() string {
	return "logger"
}

func (l *LoggerTask) Update(cron string) {

}

func (l *LoggerTask) Cron() string {
	return "@every 1s"
}

func (l *LoggerTask) Run() {
	logrus.Info("logger task running")
}
