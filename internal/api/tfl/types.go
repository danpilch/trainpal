package tfl

// LineStatus represents the status of a tube line.
type LineStatus struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	ModeName     string         `json:"modeName"`
	LineStatuses []StatusDetail `json:"lineStatuses"`
}

// StatusDetail contains the detailed status information.
type StatusDetail struct {
	StatusSeverity            int              `json:"statusSeverity"`
	StatusSeverityDescription string           `json:"statusSeverityDescription"`
	Reason                    string           `json:"reason"`
	ValidityPeriods           []ValidityPeriod `json:"validityPeriods"`
	Disruption                *Disruption      `json:"disruption,omitempty"`
}

// ValidityPeriod represents when a status is valid.
type ValidityPeriod struct {
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
	IsNow    bool   `json:"isNow"`
}

// Disruption contains disruption details.
type Disruption struct {
	Category            string `json:"category"`
	CategoryDescription string `json:"categoryDescription"`
	Description         string `json:"description"`
}

// StatusSeverity values.
const (
	StatusGoodService    = 10
	StatusMinorDelays    = 5
	StatusSevereDelays   = 4
	StatusPartSuspended  = 3
	StatusSuspended      = 2
	StatusPartClosure    = 1
	StatusPlannedClosure = 0
	StatusSpecialService = 9
	StatusServiceClosed  = 6
	StatusNoIssues       = 10
)

// IsGoodService returns true if the status indicates good service.
func (s *StatusDetail) IsGoodService() bool {
	return s.StatusSeverity == StatusGoodService
}

// HasDisruption returns true if there's any disruption.
func (s *StatusDetail) HasDisruption() bool {
	return s.StatusSeverity < StatusGoodService && s.StatusSeverity != StatusServiceClosed
}
