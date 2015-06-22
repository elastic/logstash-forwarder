package main

// Adopted from: https://github.com/pivotal-golang/bytefmt

import (
        "errors"
        "regexp"
        "strconv"
        "strings"
)

const (
        BYTE     = 1
        KILOBYTE = 1024 * BYTE
        MEGABYTE = 1024 * KILOBYTE
        GIGABYTE = 1024 * MEGABYTE
        TERABYTE = 1024 * GIGABYTE
)

var bytesPattern *regexp.Regexp = regexp.MustCompile(`(?i)^(-?\d+)([KMGT]B?|B)$`)

var invalidByteQuantityError = errors.New("Byte quantity must be a positive integer with a unit of measurement like M, MB, G, or GB")

// ToBytes takes an input string of the format: $Number$Unit
// (with trailing/leading spaces allowed) and returns the number after
// converting it to bytes. $Number should be > 0
func ToBytes(s string) (uint64, error) {
        parts := bytesPattern.FindStringSubmatch(strings.TrimSpace(s))
        if len(parts) < 3 {
                return 0, invalidByteQuantityError
        }

        value, err := strconv.ParseUint(parts[1], 10, 0)
        if err != nil || value < 1 {
                return 0, invalidByteQuantityError
        }

        var bytes uint64
        unit := strings.ToUpper(parts[2])
        switch unit[:1] {
        case "T":
                bytes = value * TERABYTE
        case "G":
                bytes = value * GIGABYTE
        case "M":
                bytes = value * MEGABYTE
        case "K":
                bytes = value * KILOBYTE
        case "B":
                bytes = value * BYTE
        }

        return bytes, nil
}
