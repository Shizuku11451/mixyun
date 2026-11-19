//go:build !(darwin && (amd64 || arm64))

package output

// DeviceRate returns 0: only Core Audio's rate is read so far.
//
// shortcut: no WASAPI mix rate or PulseAudio sink rate yet, add them when a Windows or Linux user asks.
func DeviceRate(id string) float64 { return 0 }
