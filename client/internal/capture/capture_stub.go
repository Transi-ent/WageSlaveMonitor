//go:build !windows

package capture

import "errors"
import "time"

type Shot struct {
	MonitorIndex int
	CapturedAt   time.Time
	JPEG         []byte
}

func CaptureAll(quality int) ([]Shot, error) {
	return nil, errors.New("capture is supported on windows only")
}
