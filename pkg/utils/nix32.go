package utils

const nix32Alphabet = "0123456789abcdfghijklmnpqrsvwxyz"

// EncodeNix32 encodes a 20-byte Nix store path hash using Nix's base-32 format.
//
// This mirrors Nix's BaseNix32::encode implementation. Unlike conventional
// base-32 encodings, Nix reads five-bit groups from the input in reverse order
// and uses its own alphabet.
func EncodeNix32(data [20]byte) string {
	const encodedLength = 32

	encoded := make([]byte, 0, encodedLength)
	for n := encodedLength - 1; n >= 0; n-- {
		bit := n * 5
		i := bit / 8
		j := bit % 8

		// A five-bit group may span two adjacent input bytes.
		value := uint16(data[i]) >> j
		if i < len(data)-1 {
			value |= uint16(data[i+1]) << (8 - j)
		}

		encoded = append(encoded, nix32Alphabet[value&0x1f])
	}

	return string(encoded)
}
