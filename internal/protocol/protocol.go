package protocol

import (
	"bytes"
	"encoding/json"
)

const (
	ProtocolVersion = 1
	ProbePrefix     = "goose_version_probe:v1"
)

// Wire-level batching contract. Both peers (carrier client, exit server) must
// agree on these values: if the carrier emits a 256 KB-payload frame but the
// exit server caps reassembly at 128 KB, the connection breaks mid-stream.
// Defining them once here, rather than mirroring the same numbers in two
// packages with "must match" comments, removes a class of silent-drift bugs.
const (
	// MaxFramePayload caps the bytes per frame; larger writes are chunked.
	// Raised from 128 KB once the single-seal-per-batch envelope eliminated
	// per-frame crypto overhead — fewer, larger frames now cost strictly less
	// (less length-prefix overhead, fewer Unmarshal calls).
	MaxFramePayload = 256 * 1024

	// MaxDrainFramesPerSession keeps one hot session from monopolising an
	// entire batch when many interactive sessions are active concurrently.
	MaxDrainFramesPerSession = 8

	// MaxDrainFramesPerBatch bounds total frames packed into one HTTP
	// request/response body under normal load, so very high session fan-out
	// does not produce oversized POSTs.
	MaxDrainFramesPerBatch = 48

	// BusySessionThreshold is the active-session count above which both
	// peers switch into "busy mode": larger batch caps to reduce queueing,
	// shorter coalesce windows because the next batch fills within a few ms
	// anyway. Picked empirically — mobile apps that open ~20+ parallel
	// connections (browser tabs, video apps, chat apps) hit this routinely.
	BusySessionThreshold = 24

	// MaxDrainFramesPerBatchBusy is the busy-mode batch cap. Higher than
	// the normal-mode cap to reduce per-session queueing delay when many
	// sessions are competing for batch slots, but still bounded so the
	// resulting HTTP body stays comfortably below the 32 MB client read cap
	// (144 frames × 256 KB max payload = 36 MB raw; the exit server's
	// maxResponseBytesPreEncode budget cuts that to ~22 MB in practice).
	MaxDrainFramesPerBatchBusy = 144
)

type VersionInfo struct {
	OK              bool     `json:"ok"`
	Protocol        int      `json:"protocol"`
	ServerVersion   string   `json:"server_version"`
	MaxFramePayload int      `json:"max_frame_payload"`
	Features        []string `json:"features"`
}

type VersionProbe struct {
	Type          string `json:"type"`
	ClientVersion string `json:"client_version"`
	Protocol      int    `json:"protocol"`
}

func EncodeProbePayload(clientVersion string) []byte {
	probe := VersionProbe{
		Type:          "version_probe",
		ClientVersion: clientVersion,
		Protocol:      ProtocolVersion,
	}
	b, _ := json.Marshal(probe)
	return append([]byte(ProbePrefix+"|"), b...)
}

func IsProbePayload(payload []byte) bool {
	return bytes.HasPrefix(payload, []byte(ProbePrefix+"|")) || bytes.Equal(payload, []byte(ProbePrefix))
}

func DecodeVersionInfo(payload []byte) (*VersionInfo, error) {
	var info VersionInfo
	if err := json.Unmarshal(bytes.TrimSpace(payload), &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func EncodeVersionInfo(serverVersion string, maxFramePayload int, features []string) ([]byte, error) {
	info := VersionInfo{
		OK:              true,
		Protocol:        ProtocolVersion,
		ServerVersion:   serverVersion,
		MaxFramePayload: maxFramePayload,
		Features:        features,
	}
	return json.Marshal(info)
}
