package analysis

import "time"

type HRDetector struct {
	threshold      float32
	lastPeakTime   time.Time
	refractory     time.Duration
	lastValue      float32
	initialized    bool
}

func NewHRDetector() *HRDetector {
	return &HRDetector{
		threshold:  0.6,                 // ajustable
		refractory: 200 * time.Millisecond,
	}
}

// Process devuelve BPM si detecta un nuevo latido
func (h *HRDetector) Process(value float32, ts time.Time) (int, bool) {

	if !h.initialized {
		h.initialized = true
		h.lastValue = value
		return 0, false
	}

	// detectar cruce ascendente por threshold
	if h.lastValue < h.threshold && value >= h.threshold {

		if ts.Sub(h.lastPeakTime) > h.refractory {

			if !h.lastPeakTime.IsZero() {
				rr := ts.Sub(h.lastPeakTime).Seconds()
				bpm := int(60.0 / rr)
				h.lastPeakTime = ts
				h.lastValue = value
				return bpm, true
			}

			h.lastPeakTime = ts
		}
	}

	h.lastValue = value
	return 0, false
}

