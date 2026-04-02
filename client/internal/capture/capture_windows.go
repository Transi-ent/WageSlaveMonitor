//go:build windows

package capture

import (
	"bytes"
	"image/jpeg"
	"time"

	"github.com/kbinani/screenshot"
)

type Shot struct {
	MonitorIndex int
	CapturedAt   time.Time
	JPEG         []byte
}

func CaptureAll(quality int) ([]Shot, error) {
	if quality <= 0 || quality > 100 {
		quality = 75
	}
	n := screenshot.NumActiveDisplays()
	now := time.Now().UTC()
	out := make([]Shot, 0, n)
	for i := 0; i < n; i++ {
		rect := screenshot.GetDisplayBounds(i)
		img, err := screenshot.CaptureRect(rect)
		if err != nil {
			return nil, err
		}
		buf := bytes.NewBuffer(nil)
		if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
			return nil, err
		}
		out = append(out, Shot{
			MonitorIndex: i,
			CapturedAt:   now,
			JPEG:         buf.Bytes(),
		})
	}
	return out, nil
}
