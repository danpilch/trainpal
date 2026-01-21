package monitor

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/danpilch/trainpal/internal/api/rtt"
	"github.com/danpilch/trainpal/internal/notify"
)

type TrainMonitor struct {
	rttClient *rtt.Client
	notifier  *notify.Notifier
	logger    *logrus.Logger

	mu              sync.Mutex
	notifiedDelays  map[string]int
	notifiedCancels map[string]bool
}

func NewTrainMonitor(rttClient *rtt.Client, notifier *notify.Notifier, logger *logrus.Logger) *TrainMonitor {
	return &TrainMonitor{
		rttClient:       rttClient,
		notifier:        notifier,
		logger:          logger,
		notifiedDelays:  make(map[string]int),
		notifiedCancels: make(map[string]bool),
	}
}

func (m *TrainMonitor) ResetNotificationState() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifiedDelays = make(map[string]int)
	m.notifiedCancels = make(map[string]bool)
}

// GetExpectedArrivalTime returns the expected arrival time at the destination for a given train.
func (m *TrainMonitor) GetExpectedArrivalTime(ctx context.Context, from, to, departureTime string) (time.Time, error) {
	depTime, err := parseTimeToday(departureTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing departure time: %w", err)
	}

	resp, err := m.rttClient.Search(ctx, from, to, depTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("searching for train: %w", err)
	}

	if resp == nil || len(resp.Services) == 0 {
		return time.Time{}, fmt.Errorf("no services found")
	}

	service := m.findMatchingService(resp.Services, departureTime)
	if service == nil {
		return time.Time{}, fmt.Errorf("no matching service found")
	}

	details, err := m.rttClient.GetService(ctx, service.ServiceUid, depTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("getting service details: %w", err)
	}

	var stationCodes []string
	for _, loc := range details.Locations {
		stationCodes = append(stationCodes, loc.CRS)
		if loc.CRS == to {
			arrivalStr := loc.RealtimeArrival
			if arrivalStr == "" {
				arrivalStr = loc.GbttBookedArrival
			}
			if arrivalStr == "" {
				return time.Time{}, fmt.Errorf("no arrival time found for destination")
			}

			arrivalTime, err := parseHHMM(arrivalStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("parsing arrival time: %w", err)
			}

			now := time.Now()
			return time.Date(now.Year(), now.Month(), now.Day(),
				arrivalTime.Hour(), arrivalTime.Minute(), 0, 0, time.Local), nil
		}
	}

	m.logger.WithFields(logrus.Fields{
		"service":  service.ServiceUid,
		"to":       to,
		"stations": stationCodes,
	}).Debug("destination not found in service locations")

	return time.Time{}, fmt.Errorf("destination %s not found in service", to)
}

func (m *TrainMonitor) CheckDelay(ctx context.Context, from, to, departureTime string) error {
	depTime, err := parseTimeToday(departureTime)
	if err != nil {
		return fmt.Errorf("parsing departure time: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"from":      from,
		"to":        to,
		"departure": departureTime,
	}).Info("checking train delay")

	resp, err := m.rttClient.Search(ctx, from, to, depTime)
	if err != nil {
		return fmt.Errorf("searching for train: %w", err)
	}

	if resp == nil || len(resp.Services) == 0 {
		m.logger.WithFields(logrus.Fields{
			"from":      from,
			"to":        to,
			"departure": departureTime,
		}).Warn("no services found")
		return nil
	}

	service := m.findMatchingService(resp.Services, departureTime)
	if service == nil {
		m.logger.WithField("departure", departureTime).Warn("no matching service found for departure time")
		return nil
	}

	return m.processService(service, from, to, false)
}

// CheckStatus checks train status and always sends a notification (on time or delayed).
func (m *TrainMonitor) CheckStatus(ctx context.Context, from, to, departureTime string) error {
	depTime, err := parseTimeToday(departureTime)
	if err != nil {
		return fmt.Errorf("parsing departure time: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"from":      from,
		"to":        to,
		"departure": departureTime,
	}).Info("checking train status")

	resp, err := m.rttClient.Search(ctx, from, to, depTime)
	if err != nil {
		return fmt.Errorf("searching for train: %w", err)
	}

	if resp == nil || len(resp.Services) == 0 {
		m.logger.WithFields(logrus.Fields{
			"from":      from,
			"to":        to,
			"departure": departureTime,
		}).Warn("no services found")
		return nil
	}

	service := m.findMatchingService(resp.Services, departureTime)
	if service == nil {
		m.logger.WithField("departure", departureTime).Warn("no matching service found for departure time")
		return nil
	}

	return m.processService(service, from, to, true)
}

func (m *TrainMonitor) findMatchingService(services []rtt.Service, targetTime string) *rtt.Service {
	for i := range services {
		svc := &services[i]
		bookedDep := svc.LocationDetail.GbttBookedDeparture
		if bookedDep == targetTime || strings.HasPrefix(bookedDep, targetTime) {
			return svc
		}
	}
	return nil
}

func (m *TrainMonitor) processService(svc *rtt.Service, from, to string, alwaysNotify bool) error {
	detail := &svc.LocationDetail

	if detail.DisplayAs == "CANCELLED_CALL" || detail.DisplayAs == "CANCELLED" {
		return m.handleCancellation(svc, from, to)
	}

	platform := detail.Platform
	if platform == "" {
		platform = "TBC"
	}

	var delayMins int
	if detail.RealtimeDeparture != "" && detail.GbttBookedDeparture != "" {
		delayMins = m.calculateDelay(detail.GbttBookedDeparture, detail.RealtimeDeparture)
	}

	if delayMins > 0 {
		if alwaysNotify {
			// Status update: always send delay notification
			m.logger.WithFields(logrus.Fields{
				"service":       svc.ServiceUid,
				"delay_minutes": delayMins,
				"expected":      detail.RealtimeDeparture,
				"platform":      platform,
			}).Warn("train delayed")
			return m.notifier.SendTrainDelay(svc.ServiceUid, from, to, delayMins, detail.RealtimeDeparture, platform)
		}
		// Delay check: use deduplication
		return m.handleDelay(svc, from, to, delayMins)
	}

	m.logger.WithFields(logrus.Fields{
		"service":   svc.ServiceUid,
		"scheduled": detail.GbttBookedDeparture,
		"platform":  platform,
	}).Info("train running on time")

	if alwaysNotify {
		return m.notifier.SendTrainOnTime(svc.ServiceUid, from, to, detail.GbttBookedDeparture, platform)
	}

	return nil
}

func (m *TrainMonitor) handleCancellation(svc *rtt.Service, from, to string) error {
	m.mu.Lock()
	alreadyNotified := m.notifiedCancels[svc.ServiceUid]
	if !alreadyNotified {
		m.notifiedCancels[svc.ServiceUid] = true
	}
	m.mu.Unlock()

	if alreadyNotified {
		return nil
	}

	reason := svc.LocationDetail.CancelReasonShortText
	if reason == "" {
		reason = "No reason provided"
	}

	m.logger.WithFields(logrus.Fields{
		"service": svc.ServiceUid,
		"reason":  reason,
	}).Warn("train cancelled")

	return m.notifier.SendTrainCancellation(svc.ServiceUid, from, to, reason)
}

func (m *TrainMonitor) handleDelay(svc *rtt.Service, from, to string, delayMins int) error {
	delayBucket := delayMins / 5 * 5

	m.mu.Lock()
	lastBucket := m.notifiedDelays[svc.ServiceUid]
	shouldNotify := delayBucket > lastBucket
	if shouldNotify {
		m.notifiedDelays[svc.ServiceUid] = delayBucket
	}
	m.mu.Unlock()

	if !shouldNotify {
		m.logger.WithFields(logrus.Fields{
			"service": svc.ServiceUid,
			"delay":   delayMins,
			"bucket":  delayBucket,
		}).Debug("delay already notified for this bucket")
		return nil
	}

	detail := &svc.LocationDetail
	platform := detail.Platform
	if platform == "" {
		platform = "TBC"
	}

	m.logger.WithFields(logrus.Fields{
		"service":       svc.ServiceUid,
		"delay_minutes": delayMins,
		"expected":      detail.RealtimeDeparture,
		"platform":      platform,
	}).Warn("train delayed")

	return m.notifier.SendTrainDelay(
		svc.ServiceUid,
		from, to,
		delayMins,
		detail.RealtimeDeparture,
		platform,
	)
}

func (m *TrainMonitor) calculateDelay(scheduled, actual string) int {
	schedTime, err := parseHHMM(scheduled)
	if err != nil {
		return 0
	}
	actualTime, err := parseHHMM(actual)
	if err != nil {
		return 0
	}

	diff := actualTime.Sub(schedTime)
	if diff < 0 {
		return 0
	}
	return int(diff.Minutes())
}

func (m *TrainMonitor) CheckArrival(ctx context.Context, from, to, departureTime string) (arrived bool, err error) {
	depTime, err := parseTimeToday(departureTime)
	if err != nil {
		return false, fmt.Errorf("parsing departure time: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"from":      from,
		"to":        to,
		"departure": departureTime,
	}).Info("checking train arrival")

	resp, err := m.rttClient.Search(ctx, from, to, depTime)
	if err != nil {
		return false, fmt.Errorf("searching for train: %w", err)
	}

	if resp == nil || len(resp.Services) == 0 {
		return false, nil
	}

	service := m.findMatchingService(resp.Services, departureTime)
	if service == nil {
		return false, nil
	}

	details, err := m.rttClient.GetService(ctx, service.ServiceUid, depTime)
	if err != nil {
		return false, fmt.Errorf("getting service details: %w", err)
	}

	for _, loc := range details.Locations {
		if loc.CRS == to {
			if loc.RealtimeArrivalActual {
				arrivalTime := loc.RealtimeArrival
				if arrivalTime == "" {
					arrivalTime = loc.GbttBookedArrival
				}

				m.logger.WithFields(logrus.Fields{
					"service":      service.ServiceUid,
					"station":      to,
					"arrival_time": arrivalTime,
				}).Info("train arrived")

				if err := m.notifier.SendTrainArrival(service.ServiceUid, to, arrivalTime); err != nil {
					return true, fmt.Errorf("sending arrival notification: %w", err)
				}
				return true, nil
			}
			break
		}
	}

	return false, nil
}

func parseTimeToday(timeStr string) (time.Time, error) {
	t, err := time.Parse("1504", timeStr)
	if err != nil {
		return time.Time{}, err
	}
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, time.Local), nil
}

func parseHHMM(s string) (time.Time, error) {
	s = strings.ReplaceAll(s, ":", "")
	if len(s) < 4 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", s)
	}
	h, err := strconv.Atoi(s[:2])
	if err != nil {
		return time.Time{}, err
	}
	m, err := strconv.Atoi(s[2:4])
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(0, 1, 1, h, m, 0, 0, time.UTC), nil
}
