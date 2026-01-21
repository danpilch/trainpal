# TrainPal

Monitors train delays and Northern Line status, sends Pushover notifications.

## Features

- Checks train delays at 60/45/30 minutes before departure
- Notifies on arrival at destination
- Monitors Northern Line status every 5 minutes before morning train
- Alerts on cancellations (high priority)
- Day-of-week filtering

## Environment Variables

```bash
export PUSHOVER_TOKEN="your_token"
export PUSHOVER_USER="your_user_key"
export RTT_USERNAME="your_rtt_username"
export RTT_PASSWORD="your_rtt_password"
```

## Configuration

```yaml
morning_train:
  from: "WIN"          # Station CRS code
  to: "WAT"
  departure: "0720"
  days:                # Optional, omit for every day
    - wednesday

evening_train:
  from: "WAT"
  to: "WIN"
  departure: "1635"
  days:
    - wednesday
```

## Usage

```bash
go build ./cmd/trainpal
./trainpal --config config.yaml
```

## APIs

- [RealTimeTrains](https://api.rtt.io/) - Train times and delays
- [TfL](https://api.tfl.gov.uk/) - Northern Line status
- [Pushover](https://pushover.net/) - Notifications
