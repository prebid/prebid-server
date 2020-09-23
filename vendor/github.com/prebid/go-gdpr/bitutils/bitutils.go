package bitutils

import (
	"encoding/binary"
	"fmt"
)

// ParseByte4 parses 4 bits of data from the data array, starting at the given index
func ParseByte4(data []byte, bitStartIndex uint) (byte, error) {
	startByte := bitStartIndex / 8
	bitStartOffset := bitStartIndex % 8
	if bitStartOffset < 5 {
		if uint(len(data)) < (startByte + 1) {
			return 0, fmt.Errorf("ParseByte4 expected 4 bits to start at bit %d, but the consent string was only %d bytes long", bitStartIndex, len(data))
		}
		return (data[startByte] & (0xf0 >> bitStartOffset)) >> (4 - bitStartOffset), nil
	}
	if uint(len(data)) < (startByte+2) && bitStartOffset > 4 {
		return 0, fmt.Errorf("ParseByte4 expected 4 bits to start at bit %d, but the consent string was only %d bytes long (needs second byte)", bitStartIndex, len(data))
	}

	leftBits := (data[startByte] & (0xf0 >> bitStartOffset)) << (bitStartOffset - 4)
	bitsConsumed := 8 - bitStartOffset
	overflow := 4 - bitsConsumed
	rightBits := (data[startByte+1] & (0xf0 << (4 - overflow))) >> (8 - overflow)
	return leftBits | rightBits, nil
}

// ParseByte8 parses 8 bits of data from the data array, starting at the given index
func ParseByte8(data []byte, bitStartIndex uint) (byte, error) {
	startByte := bitStartIndex / 8
	bitStartOffset := bitStartIndex % 8
	if bitStartOffset == 0 {
		if uint(len(data)) < (startByte + 1) {
			return 0, fmt.Errorf("ParseByte8 expected 8 bits to start at bit %d, but the consent string was only %d bytes long", bitStartIndex, len(data))
		}
		return data[startByte], nil
	}
	if uint(len(data)) < (startByte + 2) {
		return 0, fmt.Errorf("ParseByte8 expected 8 bitst to start at bit %d, but the consent string was only %d bytes long", bitStartIndex, len(data))
	}

	leftBits := (data[startByte] & (0xff >> bitStartOffset)) << bitStartOffset
	shiftComplement := 8 - bitStartOffset
	rightBits := (data[startByte+1] & (0xff << shiftComplement)) >> shiftComplement
	return leftBits | rightBits, nil
}

// ParseUInt12 parses 12 bits of data fromt the data array, starting at the given index
func ParseUInt12(data []byte, bitStartIndex uint) (uint16, error) {
	end := bitStartIndex + 12
	endByte := end / 8
	endOffset := end % 8

	if endOffset > 0 {
		endByte++
	}
	if uint(len(data)) < endByte {
		return 0, fmt.Errorf("ParseUInt12 expected a 12-bit int to start at bit %d, but the consent string was only %d bytes long",
			bitStartIndex, len(data))
	}

	leftByte, err := ParseByte4(data, bitStartIndex)
	if err != nil {
		return 0, fmt.Errorf("ParseUInt12 error on left byte: %s", err)
	}
	rightByte, err := ParseByte8(data, bitStartIndex+4)
	if err != nil {
		return 0, fmt.Errorf("ParseUInt12 error on right byte: %s", err)
	}
	return binary.BigEndian.Uint16([]byte{leftByte, rightByte}), nil
}

// ParseUInt16  parses a 16-bit integer from the data array, starting at the given index
func ParseUInt16(data []byte, bitStartIndex uint) (uint16, error) {
	startByte := bitStartIndex / 8
	bitStartOffset := bitStartIndex % 8
	if bitStartOffset == 0 {
		if uint(len(data)) < (startByte + 2) {
			return 0, fmt.Errorf("ParseUInt16 expected a 16-bit int to start at bit %d, but the consent string was only %d bytes long", bitStartIndex, len(data))
		}
		return binary.BigEndian.Uint16(data[startByte : startByte+2]), nil
	}
	if uint(len(data)) < (startByte + 3) {
		return 0, fmt.Errorf("ParseUInt16 expected a 16-bit int to start at bit %d, but the consent string was only %d bytes long", bitStartIndex, len(data))
	}

	leftByte, err := ParseByte8(data, bitStartIndex)
	if err != nil {
		return 0, fmt.Errorf("ParseUInt16 error on left byte: %s", err)
	}
	rightByte, err := ParseByte8(data, bitStartIndex+8)
	if err != nil {
		return 0, fmt.Errorf("ParseUInt16 error on right byte: %s", err)
	}
	return binary.BigEndian.Uint16([]byte{leftByte, rightByte}), nil
}
