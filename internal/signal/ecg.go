package signal

import "math"

// ECGSim genera una forma tipo ECG (no clínica) a fs Hz.
// Es deliberadamente simple: baseline + “QRS” gaussiano + algo de ruido.
type ECGSim struct {
	fs     float64
	phase  float64
	hrBPM  float64
	noise  float64
}

// NewECGSim fs=250, hrBPM típico 60-120, noise ~0.0-0.05
func NewECGSim(fs, hrBPM, noise float64) *ECGSim {
	return &ECGSim{fs: fs, hrBPM: hrBPM, noise: noise}
}

// Next devuelve el próximo sample (float32) y avanza el tiempo.
func (s *ECGSim) Next() float32 {
	// tiempo normalizado dentro del ciclo [0..1)
	cycleHz := s.hrBPM / 60.0
	s.phase += cycleHz / s.fs
	if s.phase >= 1.0 {
		s.phase -= 1.0
	}

	t := s.phase // 0..1

	// baseline (respiración suave)
	baseline := 0.05 * math.Sin(2*math.Pi*0.33*float64(t)) // 0.33 “Hz dentro del ciclo” ~ lento

	// P, QRS, T como gaussianas
	p := 0.08 * gauss(t, 0.18, 0.03)
	q := -0.12 * gauss(t, 0.30, 0.01)
	r := 1.00 * gauss(t, 0.32, 0.008)
	sv := -0.25 * gauss(t, 0.35, 0.012)
	tt := 0.25 * gauss(t, 0.60, 0.06)

	// ruido determinista simple (barato)
	n := s.noise * (2*fract(math.Sin(12345.678*float64(t))*9876.543) - 1)

	return float32(baseline + p + q + r + sv + tt + n)
}

func gauss(x, mu, sigma float64) float64 {
	z := (x - mu) / sigma
	return math.Exp(-0.5 * z * z)
}

func fract(x float64) float64 { return x - math.Floor(x) }

