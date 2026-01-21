package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/danpilch/trainpal/internal/config"
	"github.com/danpilch/trainpal/internal/monitor"
)

type TaskType int

const (
	TaskMorningDelayCheck TaskType = iota
	TaskEveningDelayCheck
	TaskMorningArrivalCheck
	TaskEveningArrivalCheck
	TaskNorthernLineCheck
	TaskNorthernLineSummary
	TaskMorningStatusUpdate
	TaskEveningStatusUpdate
)

type Task struct {
	Type      TaskType
	Time      time.Time
	Executed  bool
	Repeating bool
}

type Scheduler struct {
	cfg          *config.Config
	trainMonitor *monitor.TrainMonitor
	tubeMonitor  *monitor.TubeMonitor
	logger       *logrus.Logger

	mu             sync.Mutex
	tasks          []Task
	currentDay     int
	arrivalPolling map[TaskType]bool
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

func NewScheduler(
	cfg *config.Config,
	trainMonitor *monitor.TrainMonitor,
	tubeMonitor *monitor.TubeMonitor,
	logger *logrus.Logger,
) *Scheduler {
	return &Scheduler{
		cfg:            cfg,
		trainMonitor:   trainMonitor,
		tubeMonitor:    tubeMonitor,
		logger:         logger,
		arrivalPolling: make(map[TaskType]bool),
		stopCh:         make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.wg.Add(1)
	go s.run(ctx)
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	s.setupDailyTasks()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("scheduler stopped: context cancelled")
			return
		case <-s.stopCh:
			s.logger.Info("scheduler stopped: stop signal received")
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	now := time.Now()

	if now.Day() != s.currentDay {
		s.logger.Info("day changed, resetting tasks")
		s.trainMonitor.ResetNotificationState()
		s.tubeMonitor.ResetNotificationState()
		s.setupDailyTasks()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.tasks {
		task := &s.tasks[i]
		if task.Executed && !task.Repeating {
			continue
		}

		if s.isWithinWindow(task.Time, now, 2*time.Minute) {
			s.executeTask(ctx, task)
		}
	}
}

func (s *Scheduler) isWithinWindow(taskTime, now time.Time, window time.Duration) bool {
	diff := now.Sub(taskTime)
	return diff >= 0 && diff < window
}

func (s *Scheduler) setupDailyTasks() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.currentDay = now.Day()
	s.tasks = nil
	s.arrivalPolling = make(map[TaskType]bool)

	morningActive := s.cfg.MorningTrain.IsActiveToday()
	eveningActive := s.cfg.EveningTrain.IsActiveToday()

	if !morningActive && !eveningActive {
		s.logger.WithField("weekday", now.Weekday().String()).Info("no trains scheduled for today")
		return
	}

	var morningDep, eveningDep time.Time
	var err error

	if morningActive {
		morningDep, err = s.cfg.MorningTrain.DepartureTime()
		if err != nil {
			s.logger.WithField("error", err).Error("failed to parse morning departure time")
			morningActive = false
		}
	}

	if eveningActive {
		eveningDep, err = s.cfg.EveningTrain.DepartureTime()
		if err != nil {
			s.logger.WithField("error", err).Error("failed to parse evening departure time")
			eveningActive = false
		}
	}

	if morningActive {
		// Delay checks (only notify on delay)
		s.tasks = append(s.tasks,
			Task{Type: TaskMorningDelayCheck, Time: morningDep.Add(-60 * time.Minute)},
			Task{Type: TaskMorningDelayCheck, Time: morningDep.Add(-45 * time.Minute)},
			Task{Type: TaskMorningDelayCheck, Time: morningDep.Add(-30 * time.Minute)},
			Task{Type: TaskMorningDelayCheck, Time: morningDep.Add(-15 * time.Minute)},
		)

		// Status updates at 60m and 30m (always notify on-time or delay)
		s.tasks = append(s.tasks,
			Task{Type: TaskMorningStatusUpdate, Time: morningDep.Add(-60 * time.Minute)},
			Task{Type: TaskMorningStatusUpdate, Time: morningDep.Add(-30 * time.Minute)},
		)

		s.tasks = append(s.tasks,
			Task{Type: TaskMorningArrivalCheck, Time: morningDep.Add(70 * time.Minute), Repeating: true},
		)

		startTime := morningDep.Add(-60 * time.Minute)
		for t := startTime; !t.After(morningDep); t = t.Add(5 * time.Minute) {
			s.tasks = append(s.tasks, Task{Type: TaskNorthernLineCheck, Time: t})
		}

		// Schedule status summary 15 mins before arrival
		arrivalTime, err := s.trainMonitor.GetExpectedArrivalTime(
			context.Background(),
			s.cfg.MorningTrain.From,
			s.cfg.MorningTrain.To,
			s.cfg.MorningTrain.Departure,
		)
		if err != nil {
			s.logger.WithField("error", err).Warn("failed to get arrival time, skipping status summary")
		} else {
			summaryTime := arrivalTime.Add(-15 * time.Minute)
			s.tasks = append(s.tasks, Task{Type: TaskNorthernLineSummary, Time: summaryTime})
			s.logger.WithFields(logrus.Fields{
				"arrival":      arrivalTime.Format("15:04"),
				"summary_time": summaryTime.Format("15:04"),
			}).Info("scheduled northern line status summary")
		}
	}

	if eveningActive {
		// Delay checks (only notify on delay)
		s.tasks = append(s.tasks,
			Task{Type: TaskEveningDelayCheck, Time: eveningDep.Add(-60 * time.Minute)},
			Task{Type: TaskEveningDelayCheck, Time: eveningDep.Add(-45 * time.Minute)},
			Task{Type: TaskEveningDelayCheck, Time: eveningDep.Add(-30 * time.Minute)},
			Task{Type: TaskEveningDelayCheck, Time: eveningDep.Add(-15 * time.Minute)},
		)

		// Status updates at 60m and 30m (always notify on-time or delay)
		s.tasks = append(s.tasks,
			Task{Type: TaskEveningStatusUpdate, Time: eveningDep.Add(-60 * time.Minute)},
			Task{Type: TaskEveningStatusUpdate, Time: eveningDep.Add(-30 * time.Minute)},
		)

		s.tasks = append(s.tasks,
			Task{Type: TaskEveningArrivalCheck, Time: eveningDep.Add(70 * time.Minute), Repeating: true},
		)
	}

	s.logger.WithFields(logrus.Fields{
		"weekday":        now.Weekday().String(),
		"morning_active": morningActive,
		"evening_active": eveningActive,
		"total_tasks":    len(s.tasks),
	}).Info("daily tasks scheduled")
}

func (s *Scheduler) executeTask(ctx context.Context, task *Task) {
	s.logger.WithFields(logrus.Fields{
		"type":           task.Type,
		"scheduled_time": task.Time.Format("15:04"),
	}).Debug("executing task")

	var err error

	switch task.Type {
	case TaskMorningDelayCheck:
		err = s.trainMonitor.CheckDelay(ctx,
			s.cfg.MorningTrain.From,
			s.cfg.MorningTrain.To,
			s.cfg.MorningTrain.Departure)

	case TaskEveningDelayCheck:
		err = s.trainMonitor.CheckDelay(ctx,
			s.cfg.EveningTrain.From,
			s.cfg.EveningTrain.To,
			s.cfg.EveningTrain.Departure)

	case TaskMorningArrivalCheck:
		arrived, checkErr := s.trainMonitor.CheckArrival(ctx,
			s.cfg.MorningTrain.From,
			s.cfg.MorningTrain.To,
			s.cfg.MorningTrain.Departure)
		err = checkErr
		if arrived {
			task.Repeating = false
		} else {
			task.Time = task.Time.Add(5 * time.Minute)
		}

	case TaskEveningArrivalCheck:
		arrived, checkErr := s.trainMonitor.CheckArrival(ctx,
			s.cfg.EveningTrain.From,
			s.cfg.EveningTrain.To,
			s.cfg.EveningTrain.Departure)
		err = checkErr
		if arrived {
			task.Repeating = false
		} else {
			task.Time = task.Time.Add(5 * time.Minute)
		}

	case TaskNorthernLineCheck:
		err = s.tubeMonitor.CheckStatus(ctx)

	case TaskNorthernLineSummary:
		err = s.tubeMonitor.SendStatusSummary(ctx)

	case TaskMorningStatusUpdate:
		err = s.trainMonitor.CheckStatus(ctx,
			s.cfg.MorningTrain.From,
			s.cfg.MorningTrain.To,
			s.cfg.MorningTrain.Departure)

	case TaskEveningStatusUpdate:
		err = s.trainMonitor.CheckStatus(ctx,
			s.cfg.EveningTrain.From,
			s.cfg.EveningTrain.To,
			s.cfg.EveningTrain.Departure)
	}

	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"type":  task.Type,
			"error": err,
		}).Error("task execution failed")
	}

	if !task.Repeating {
		task.Executed = true
	}
}
