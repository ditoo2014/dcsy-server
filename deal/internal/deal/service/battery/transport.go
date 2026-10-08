package battery

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	transportSignedDecimal   = "1"
	transportBinaryBit       = "2"
	transportUnsignedDecimal = "3"
)

// CalcChecksum follows the TTDC ASCII checksum: sum VER through INFO as ASCII,
// modulo 65536, then two's-complement it.
func CalcChecksum(frame string) string {
	end := len(frame) - 4
	if end <= 1 {
		return "0000"
	}
	dataPart := frame[1:end]
	sum := 0
	for i := 0; i < len(dataPart); i++ {
		sum += int(dataPart[i])
	}
	sum %= 65536
	return fmt.Sprintf("%04X", ((^sum)+1)&0xFFFF)
}

func IsNormalData(data string) bool {
	return len(data) == 2 && strings.EqualFold(data, "00")
}

// GetLength returns the INFO length used by the current TTDC frames. The high
// nibble is a length marker, so only the low 12 bits are meaningful here.
func GetLength(data string) int {
	if len(data) != 4 {
		return 0
	}
	raw, err := strconv.ParseUint(data, 16, 16)
	if err != nil {
		return 0
	}
	return int(raw & 0x0FFF)
}

func HexToUnsignedDecimal(data string) (uint64, error) {
	return strconv.ParseUint(data, 16, 64)
}

func HexToSignedDecimal(data string) (int64, error) {
	if len(data)%2 != 0 {
		return 0, fmt.Errorf("invalid hex length")
	}
	bytes := len(data) / 2
	bits := uint(bytes * 8)
	if bits == 0 || bits > 63 {
		return 0, fmt.Errorf("unsupported signed width: %d", bits)
	}
	u, err := strconv.ParseUint(data, 16, 64)
	if err != nil {
		return 0, err
	}
	signBit := uint64(1) << (bits - 1)
	if u&signBit == 0 {
		return int64(u), nil
	}
	return int64(u) - int64(uint64(1)<<bits), nil
}

func HexBit(data string, bit int) (int, error) {
	if bit < 0 {
		return 0, fmt.Errorf("invalid bit index: %d", bit)
	}
	u, err := strconv.ParseUint(data, 16, 64)
	if err != nil {
		return 0, err
	}
	bits := len(data) * 4
	if bit >= bits {
		return 0, fmt.Errorf("bit index out of range: %d >= %d", bit, bits)
	}
	return int((u >> uint(bit)) & 1), nil
}

// decodeTransport keeps the battery parser's transport encoding isolated from
// older dicloud TTDC transport formats. The address is an INFO hex-string
// offset, not a protocol byte index.
func decodeTransport(info string, transport string) (float64, bool) {
	parts := strings.Fields(transport)
	if len(parts) < 3 {
		return 0, false
	}
	addr, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, false
	}
	byteCount, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, false
	}
	end := addr + byteCount*2
	if addr < 0 || byteCount <= 0 || end > len(info) {
		return 0, false
	}
	raw := info[addr:end]
	switch parts[2] {
	case transportSignedDecimal:
		value, err := HexToSignedDecimal(raw)
		return float64(value), err == nil
	case transportUnsignedDecimal:
		value, err := HexToUnsignedDecimal(raw)
		return float64(value), err == nil
	case transportBinaryBit:
		if len(parts) < 4 {
			return 0, false
		}
		bit, err := strconv.Atoi(parts[3])
		if err != nil {
			return 0, false
		}
		value, err := HexBit(raw, bit)
		return float64(value), err == nil
	default:
		return 0, false
	}
}
