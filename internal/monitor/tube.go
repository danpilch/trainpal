package monitor

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/danpilch/trainpal/internal/api/tfl"
	"github.com/danpilch/trainpal/internal/notify"
)

type TubeMonitor struct {
	tflClient *tfl.Client
	notifier  *notify.Notifier
	logger    *logrus.Logger

	mu               sync.Mutex
	lastStatus       string
	lastStatusReason string
}

func NewTubeMonitor(tflClient *tfl.Client, notifier *notify.Notifier, logger *logrus.Logger) *TubeMonitor {
	return &TubeMonitor{
		tflClient: tflClient,
		notifier:  notifier,
		logger:    logger,
	}
}

func (m *TubeMonitor) ResetNotificationState() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastStatus = ""
	m.lastStatusReason = ""
}

func (m *TubeMonitor) CheckStatus(ctx context.Context) error {
	status, err := m.tflClient.GetNorthernLineStatus(ctx)
	if err != nil {
		return err
	}

	if len(status.LineStatuses) == 0 {
		m.logger.Warn("no status information available for Northern Line")
		return nil
	}

	currentStatus := status.LineStatuses[0]
	statusDesc := currentStatus.StatusSeverityDescription
	reason := currentStatus.Reason

	m.mu.Lock()
	statusChanged := m.lastStatus != statusDesc || m.lastStatusReason != reason
	isFirstCheck := m.lastStatus == ""
	m.lastStatus = statusDesc
	m.lastStatusReason = reason
	m.mu.Unlock()

	m.logger.WithFields(logrus.Fields{
		"status":   statusDesc,
		"severity": currentStatus.StatusSeverity,
		"reason":   reason,
	}).Info("northern line status")

	if !statusChanged {
		return nil
	}

	if isFirstCheck && currentStatus.IsGoodService() {
		return nil
	}

	if currentStatus.HasDisruption() || (!isFirstCheck && statusChanged) {
		if reason == "" {
			reason = "No additional details"
		}
		m.logger.WithFields(logrus.Fields{
			"status": statusDesc,
			"reason": reason,
		}).Warn("northern line disruption detected")
		return m.notifier.SendTubeDisruption(statusDesc, reason)
	}

	return nil
}

func (m *TubeMonitor) SendStatusSummary(ctx context.Context) error {
	status, err := m.tflClient.GetNorthernLineStatus(ctx)
	if err != nil {
		return err
	}

	if len(status.LineStatuses) == 0 {
		return m.notifier.SendTubeStatus("Unknown", "Unable to retrieve status")
	}

	currentStatus := status.LineStatuses[0]
	statusDesc := currentStatus.StatusSeverityDescription
	reason := currentStatus.Reason

	m.logger.WithFields(logrus.Fields{
		"status": statusDesc,
		"reason": reason,
	}).Info("sending northern line status summary")

	return m.notifier.SendTubeStatus(statusDesc, reason)
}
