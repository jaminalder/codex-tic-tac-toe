package app

import (
    "crypto/rand"
    "fmt"
)

// newUUID generates a UUIDv4 string without external deps.
func newUUID() string {
    var b [16]byte
    if _, err := rand.Read(b[:]); err != nil {
        // extremely unlikely; fall back to random-like bytes left as zero
    }
    // Set version (4) and variant (RFC 4122)
    b[6] = (b[6] & 0x0f) | 0x40
    b[8] = (b[8] & 0x3f) | 0x80
    return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
        uint32(b[0])<<24|uint32(b[1])<<16|uint32(b[2])<<8|uint32(b[3]),
        uint16(b[4])<<8|uint16(b[5]),
        uint16(b[6])<<8|uint16(b[7]),
        uint16(b[8])<<8|uint16(b[9]),
        uint64(b[10])<<40|uint64(b[11])<<32|uint64(b[12])<<24|uint64(b[13])<<16|uint64(b[14])<<8|uint64(b[15]),
    )
}

