package battery

import (
	"encoding/hex"
	"strconv"
	"strings"
	"sync"
	"time"
)

type gatewayAssembler struct {
	buffer         string
	pendingHeaders []string
	lastSeen       time.Time
}

var (
	assemblerMu    sync.Mutex
	assemblerStore = make(map[string]*gatewayAssembler)
)

const (
	maxAssemblerBuffer  = 32 * 1024
	maxProtocolInfoLen  = 4094
	assemblerStateTTL   = 10 * time.Minute
	minProtocolFrameLen = 17
	maxPendingHeaders   = 32
)

// AssemblePackets pairs a TTDC request frame with its response frame. The
// state key includes battery kind and gateway ID, so sodium and lithium-sulfur
// frames cannot be accidentally paired with each other.
func AssemblePackets(topic, chunk string) []string {
	topicKind, gatewayID, ok := ParseTopic(topic)
	if !ok {
		return nil
	}
	key := string(topicKind) + ":" + gatewayID
	normalized := normalizeChunk(chunk)
	if normalized == "" {
		return nil
	}

	assemblerMu.Lock()
	defer assemblerMu.Unlock()

	state, exists := assemblerStore[key]
	if !exists {
		state = &gatewayAssembler{}
		assemblerStore[key] = state
	}
	state.lastSeen = time.Now()
	state.buffer += normalized
	if len(state.buffer) > maxAssemblerBuffer {
		state.buffer = state.buffer[len(state.buffer)-maxAssemblerBuffer:]
	}

	frames := extractFrames(&state.buffer)
	if len(frames) == 0 {
		cleanupAssemblers(time.Now())
		return nil
	}

	out := make([]string, 0, len(frames))
	for _, frame := range frames {
		cid2 := frame[7:9]
		if cid2 == "00" {
			if len(state.pendingHeaders) > 0 {
				out = append(out, state.pendingHeaders[0]+"\r"+frame)
				state.pendingHeaders = state.pendingHeaders[1:]
			}
			continue
		}
		state.pendingHeaders = append(state.pendingHeaders, frame)
		if len(state.pendingHeaders) > maxPendingHeaders {
			state.pendingHeaders = state.pendingHeaders[len(state.pendingHeaders)-maxPendingHeaders:]
		}
	}

	cleanupAssemblers(time.Now())
	return out
}

func normalizeChunk(chunk string) string {
	chunk = strings.ToUpper(chunk)
	var b strings.Builder
	b.Grow(len(chunk))
	for i := 0; i < len(chunk); i++ {
		c := chunk[i]
		if c == '~' || c == '\r' {
			b.WriteByte(c)
			continue
		}
		if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') {
			b.WriteByte(c)
		}
	}
	return b.String()
}

func extractFrames(buffer *string) []string {
	src := *buffer
	frames := make([]string, 0, 4)

	for {
		start := strings.IndexByte(src, '~')
		if start < 0 {
			if len(src) > minProtocolFrameLen {
				src = src[len(src)-minProtocolFrameLen:]
			}
			break
		}
		if start > 0 {
			src = src[start:]
		}
		if len(src) < minProtocolFrameLen {
			break
		}

		lengthRaw, err := strconv.ParseUint(src[9:13], 16, 16)
		if err != nil {
			src = src[1:]
			continue
		}
		infoLen := int(lengthRaw & 0x0FFF)
		if infoLen < 0 || infoLen > maxProtocolInfoLen {
			src = src[1:]
			continue
		}
		frameLen := minProtocolFrameLen + infoLen
		if len(src) < frameLen {
			break
		}

		frame := src[:frameLen]
		if _, err = hex.DecodeString(frame[1:]); err != nil {
			src = src[1:]
			continue
		}
		if !strings.EqualFold(frame[len(frame)-4:], CalcChecksum(frame)) {
			src = src[1:]
			continue
		}

		frames = append(frames, frame)
		src = src[frameLen:]
		if len(src) > 0 && src[0] == '\r' {
			src = src[1:]
		}
	}

	*buffer = src
	return frames
}

func cleanupAssemblers(now time.Time) {
	for key, state := range assemblerStore {
		if now.Sub(state.lastSeen) <= assemblerStateTTL {
			continue
		}
		delete(assemblerStore, key)
	}
}
