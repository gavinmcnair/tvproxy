package tvsatipscan

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SignalInfo holds all tuner metadata from a SAT>IP DESCRIBE response.
type SignalInfo struct {
	// Signal quality
	Lock    bool `json:"lock"`
	Level   int  `json:"level"`   // 0–255 signal strength
	Quality int  `json:"quality"` // 0–15 SNR
	BER     int  `json:"ber"`

	// Tuner identity
	FeID int `json:"fe_id"` // physical frontend number

	// Tune parameters (confirmed from server, not just URL)
	FreqMHz float64 `json:"freq_mhz"`
	BwMHz   int     `json:"bw_mhz"`
	Msys    string  `json:"msys"`    // dvbt/dvbt2/dvbc/dvbs/dvbs2
	Mtype   string  `json:"mtype"`   // 256qam/64qam/8psk/etc
	PLPID   string  `json:"plp_id"`  // DVB-T2 PLP
	T2ID    string  `json:"t2_id"`   // DVB-T2 system ID

	// Stream info
	BitratKbps int  `json:"bitrate_kbps"` // b=AS: transport stream bitrate
	Active     bool `json:"active"`       // a=sendonly vs a=inactive

	// Server identity
	Server string `json:"server"` // RTSP Server header
}

// LevelPct returns signal level as a 0–100 percentage.
func (s *SignalInfo) LevelPct() int {
	if s.Level == 0 {
		return 0
	}
	return s.Level * 100 / 255
}

// QualityPct returns signal quality as a 0–100 percentage.
func (s *SignalInfo) QualityPct() int {
	if s.Quality == 0 {
		return 0
	}
	return s.Quality * 100 / 15
}

// QuerySignal sends an RTSP DESCRIBE to the SAT>IP server and parses all
// available tuner metadata from the SDP response. It does not allocate a tuner.
func QuerySignal(rtspURL string, timeout time.Duration) (*SignalInfo, error) {
	host := extractHost(rtspURL)
	if host == "" {
		return nil, fmt.Errorf("cannot extract host from %q", rtspURL)
	}

	c, err := dialRTSP(host, timeout)
	if err != nil {
		return nil, err
	}
	defer c.close()
	c.conn.SetDeadline(time.Now().Add(timeout)) //nolint

	resp, err := c.send("DESCRIBE", rtspURL, map[string]string{"Accept": "application/sdp"}, nil)
	if err != nil {
		return nil, err
	}
	if resp.status != 200 {
		return nil, fmt.Errorf("DESCRIBE returned %d", resp.status)
	}

	info := parseTunerSDP(string(resp.body))
	if info == nil {
		return nil, nil
	}
	if sv, ok := resp.headers["server"]; ok {
		info.Server = sv
	}
	return info, nil
}

// extractHost returns "host:port" from an rtsp:// URL, defaulting to port 554.
func extractHost(u string) string {
	u = strings.TrimPrefix(u, "rtsp://")
	u = strings.TrimPrefix(u, "rtsps://")
	host := strings.SplitN(u, "/", 2)[0]
	host = strings.SplitN(host, "?", 2)[0]
	if !strings.Contains(host, ":") {
		host += ":554"
	}
	return host
}

// parseTunerSDP extracts all SAT>IP tuner fields from an SDP body.
//
// Example fmtp line:
//
//	a=fmtp:33 ver=1.1;tuner=1,255,1,15,546.00,8,dvbt2,,256qam,,,,0,0;pids=0
//
// Positional tuner fields: fe,level,lock,quality,freq,bw,msys,plp,mtype,t2id,...
func parseTunerSDP(sdp string) *SignalInfo {
	info := &SignalInfo{}
	found := false

	for _, line := range strings.Split(sdp, "\n") {
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "b=AS:"):
			if v, err := strconv.Atoi(strings.TrimPrefix(line, "b=AS:")); err == nil {
				info.BitratKbps = v
			}

		case line == "a=sendonly":
			info.Active = true

		case strings.HasPrefix(line, "a=fmtp:"):
			params := strings.SplitN(line, " ", 2)
			if len(params) < 2 {
				continue
			}
			for _, kv := range strings.Split(params[1], ";") {
				if !strings.HasPrefix(kv, "tuner=") {
					continue
				}
				fields := strings.Split(strings.TrimPrefix(kv, "tuner="), ",")
				if len(fields) < 4 {
					continue
				}
				found = true
				info.FeID, _ = strconv.Atoi(fields[0])
				info.Level, _ = strconv.Atoi(fields[1])
				if lockVal, _ := strconv.Atoi(fields[2]); lockVal == 1 {
					info.Lock = true
				}
				info.Quality, _ = strconv.Atoi(fields[3])
				if len(fields) > 4 {
					if f, err := strconv.ParseFloat(fields[4], 64); err == nil {
						info.FreqMHz = f
					}
				}
				if len(fields) > 5 {
					info.BwMHz, _ = strconv.Atoi(fields[5])
				}
				if len(fields) > 6 {
					info.Msys = fields[6]
				}
				// fields[7] is often empty or PLP; fields[8] is mtype on some devices
				// Handle both orderings by checking for known msys/mtype strings
				for i := 7; i < len(fields); i++ {
					f := strings.TrimSpace(fields[i])
					if f == "" {
						continue
					}
					switch {
					case isModulationType(f):
						info.Mtype = f
					case looksLikePLP(f):
						info.PLPID = f
					}
				}
			}
		}
	}

	if !found {
		return nil
	}
	return info
}

func isModulationType(s string) bool {
	switch s {
	case "qpsk", "8psk", "16apsk", "32apsk",
		"16qam", "32qam", "64qam", "128qam", "256qam",
		"8vsb", "16vsb":
		return true
	}
	return false
}

func looksLikePLP(s string) bool {
	// PLP IDs are small integers; avoid misidentifying zero-valued extras
	if _, err := strconv.Atoi(s); err == nil {
		return true
	}
	return false
}
