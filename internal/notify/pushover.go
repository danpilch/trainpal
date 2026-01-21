package notify

import (
	"fmt"

	"github.com/gregdel/pushover"
	"github.com/sirupsen/logrus"
)

const (
	PriorityNormal = 0
	PriorityHigh   = 1
)

type Notifier struct {
	app       *pushover.Pushover
	recipient *pushover.Recipient
	logger    *logrus.Logger
}

func NewNotifier(token, userKey string, logger *logrus.Logger) *Notifier {
	return &Notifier{
		app:       pushover.New(token),
		recipient: pushover.NewRecipient(userKey),
		logger:    logger,
	}
}

func (n *Notifier) Send(title, message string) error {
	return n.SendWithPriority(title, message, PriorityNormal)
}

func (n *Notifier) SendWithPriority(title, message string, priority int) error {
	msg := pushover.NewMessageWithTitle(message, title)
	msg.Priority = priority

	resp, err := n.app.SendMessage(msg, n.recipient)
	if err != nil {
		return fmt.Errorf("sending pushover notification: %w", err)
	}

	n.logger.WithFields(logrus.Fields{
		"title":      title,
		"status":     resp.Status,
		"request_id": resp.ID,
	}).Debug("notification sent")

	return nil
}

func (n *Notifier) SendTrainDelay(trainID, from, to string, delayMinutes int, expectedTime, platform string) error {
	title := "Train Delay Alert"
	body := fmt.Sprintf("Train %s from %s to %s is delayed by %d minutes.\nExpected: %s, Platform: %s",
		trainID, from, to, delayMinutes, expectedTime, platform)
	return n.SendWithPriority(title, body, PriorityHigh)
}

func (n *Notifier) SendTrainOnTime(trainID, from, to, departureTime, platform string) error {
	title := "Train Status"
	body := fmt.Sprintf("Train %s from %s to %s is running on time.\nDeparture: %s, Platform: %s",
		trainID, from, to, departureTime, platform)
	return n.Send(title, body)
}

func (n *Notifier) SendTrainArrival(trainID, station, arrivalTime string) error {
	title := "Train Arrival"
	body := fmt.Sprintf("Train %s has arrived at %s at %s", trainID, station, arrivalTime)
	return n.Send(title, body)
}

func (n *Notifier) SendTrainDeparture(trainID, from, to, departureTime, platform string) error {
	title := "Train Departed"
	body := fmt.Sprintf("Train %s from %s to %s has departed at %s from Platform %s",
		trainID, from, to, departureTime, platform)
	return n.Send(title, body)
}

func (n *Notifier) SendTrainCancellation(trainID, from, to, reason string) error {
	title := "Train Cancellation Alert"
	body := fmt.Sprintf("Train %s from %s to %s has been CANCELLED.\nReason: %s",
		trainID, from, to, reason)
	return n.SendWithPriority(title, body, PriorityHigh)
}

func (n *Notifier) SendTubeDisruption(status, reason string) error {
	title := "Tube Disruption Alert"
	body := fmt.Sprintf("Northern Line: %s\n%s", status, reason)
	return n.SendWithPriority(title, body, PriorityHigh)
}

func (n *Notifier) SendTubeStatus(status, reason string) error {
	title := "Northern Line Status"
	body := status
	if reason != "" {
		body = fmt.Sprintf("%s\n%s", status, reason)
	}
	return n.Send(title, body)
}
