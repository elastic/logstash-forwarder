package main

import (
        "log"
        "testing"
        "strings"
)

// helper for functions testing valid inputs
func bytesfmtTestHelperValid(t *testing.T, input string, expOut uint64) {
        bytes, err := ToBytes(input)
        if err != nil {
            t.Fatal("Should have succeeded", err, "Input:", input)
        }
        if bytes != expOut {
            t.Fatal("Input:", input, "Output:", bytes, "Expected:", expOut)
        }
}

// helper for functions testing invalid inputs
func bytesfmtTestHelperInvalid(t *testing.T, input string) {
        _, err := ToBytes(input)
        if err == nil {
            t.Fatal("Should have failed but didn't! Input:", input)
        }
        if !strings.Contains(err.Error(), "unit of measurement") {
            t.Fatal("Unexpected error encountered:", err)
        }
}

// parses byte amounts with short units (e.g. M, G)
func TestParseBytesWithShortUnits(t *testing.T) {
        log.Println("\n-- TestParseBytesWithShortUnits -- ")
        bytesfmtTestHelperValid(t, "5B", 5)
        bytesfmtTestHelperValid(t, "5K", 5120)
        bytesfmtTestHelperValid(t, "5M", 5242880)
        bytesfmtTestHelperValid(t, "2G", 2147483648)
        bytesfmtTestHelperValid(t, "3T", 3298534883328)
}

// parses byte amounts with long units (e.g MB, GB)
func TestParseBytesWithLongUnits(t *testing.T) {
        log.Println("\n-- TestParseBytesWithLongUnits -- ")
        bytesfmtTestHelperValid(t, "5MB", 5242880)
        bytesfmtTestHelperValid(t, "5mb", 5242880)
        bytesfmtTestHelperValid(t, "2GB", 2147483648)
        bytesfmtTestHelperValid(t, "3TB", 3298534883328)
}

// check for error when the unit is missing
func TestParseBytesWithoutUnits(t *testing.T) {
        log.Println("\n-- TestParseBytesWithoutUnits --")
        bytesfmtTestHelperInvalid(t, "5")
}

// check for error when the unit is unrecognized
func TestParseBytesUnkownUnits(t *testing.T) {
        log.Println("\n-- TestParseBytesUnkownUnits --")
        bytesfmtTestHelperInvalid(t, "5MBB")
        bytesfmtTestHelperInvalid(t, "5BB")
}

// allow whitespace before and after the value
func TestParseBytesWithWhiteSpace(t *testing.T) {
        log.Println("\n-- TestParseBytesWithWhiteSpace --")
        bytesfmtTestHelperValid(t, "\t\n\r 5MB", 5242880)
        bytesfmtTestHelperInvalid(t, "5  TB")
}

// check for error when input is negative
func TestParseBytesNegativeValue(t *testing.T) {
        log.Println("\n-- TestParseBytesNegavitveValue --")
        bytesfmtTestHelperInvalid(t, "-5MB")
}

// check for error when input is 0
func TestParseBytesZeros(t *testing.T) {
        log.Println("\n-- TestParseBytesZeros --")
        bytesfmtTestHelperInvalid(t, "0TB")
        bytesfmtTestHelperInvalid(t, "0M")
}

// check for error when the input is empty
func TestParseBytesEmptyInput(t *testing.T) {
        log.Println("\n-- TestParseBytesEmptyInput --")
        bytesfmtTestHelperInvalid(t, "")
}
