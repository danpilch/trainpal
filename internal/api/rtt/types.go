package rtt

// SearchResponse represents the response from the RTT search endpoint.
type SearchResponse struct {
	Location          Location  `json:"location"`
	Filter            Location  `json:"filter"`
	Services          []Service `json:"services"`
	RealtimeAvailable bool      `json:"realtimeAvailable"`
}

// Location represents a station location.
type Location struct {
	Name string `json:"name"`
	CRS  string `json:"crs"`
}

// Service represents a train service.
type Service struct {
	ServiceUid     string         `json:"serviceUid"`
	RunDate        string         `json:"runDate"`
	ServiceType    string         `json:"serviceType"`
	AtocCode       string         `json:"atocCode"`
	AtocName       string         `json:"atocName"`
	LocationDetail LocationDetail `json:"locationDetail"`
}

// LocationDetail contains timing and platform information for a location.
type LocationDetail struct {
	RealtimeActivated       bool           `json:"realtimeActivated"`
	Origin                  []LocationName `json:"origin"`
	Destination             []LocationName `json:"destination"`
	GbttBookedArrival       string         `json:"gbttBookedArrival"`
	GbttBookedDeparture     string         `json:"gbttBookedDeparture"`
	RealtimeArrival         string         `json:"realtimeArrival"`
	RealtimeDeparture       string         `json:"realtimeDeparture"`
	RealtimeArrivalActual   bool           `json:"realtimeArrivalActual"`
	RealtimeDepartureActual bool           `json:"realtimeDepartureActual"`
	Platform                string         `json:"platform"`
	PlatformConfirmed       bool           `json:"platformConfirmed"`
	PlatformChanged         bool           `json:"platformChanged"`
	DisplayAs               string         `json:"displayAs"`
	CancelReasonCode        string         `json:"cancelReasonCode"`
	CancelReasonShortText   string         `json:"cancelReasonShortText"`
	CancelReasonLongText    string         `json:"cancelReasonLongText"`
}

// LocationName represents a named location with CRS code.
type LocationName struct {
	Description string `json:"description"`
	PublicTime  string `json:"publicTime"`
	CRS         string `json:"crs"`
}

// ServiceDetailResponse represents the detailed service response.
type ServiceDetailResponse struct {
	ServiceUid  string            `json:"serviceUid"`
	RunDate     string            `json:"runDate"`
	ServiceType string            `json:"serviceType"`
	AtocCode    string            `json:"atocCode"`
	AtocName    string            `json:"atocName"`
	Origin      []LocationName    `json:"origin"`
	Destination []LocationName    `json:"destination"`
	Locations   []ServiceLocation `json:"locations"`
}

// ServiceLocation represents a location in the service journey.
type ServiceLocation struct {
	RealtimeActivated       bool   `json:"realtimeActivated"`
	Tiploc                  string `json:"tiploc"`
	CRS                     string `json:"crs"`
	Description             string `json:"description"`
	GbttBookedArrival       string `json:"gbttBookedArrival"`
	GbttBookedDeparture     string `json:"gbttBookedDeparture"`
	RealtimeArrival         string `json:"realtimeArrival"`
	RealtimeDeparture       string `json:"realtimeDeparture"`
	RealtimeArrivalActual   bool   `json:"realtimeArrivalActual"`
	RealtimeDepartureActual bool   `json:"realtimeDepartureActual"`
	Platform                string `json:"platform"`
	PlatformConfirmed       bool   `json:"platformConfirmed"`
	PlatformChanged         bool   `json:"platformChanged"`
	DisplayAs               string `json:"displayAs"`
	CancelReasonCode        string `json:"cancelReasonCode"`
	CancelReasonShortText   string `json:"cancelReasonShortText"`
	CancelReasonLongText    string `json:"cancelReasonLongText"`
}
