package sastopo

import (
	"bytes"
	"encoding/hex"
	"unicode"
)

func itob(i int) bool {
	if i == 0 {
		return false
	}
	return true
}

// trimPoints loops through a []byte looking for left trim and right trim points
// for trimming 0x00 (null) and 0x20 (ascii space)
func trimPoints(line []byte) (start int, stop int) {

	//left trim point
	for i := 0; i < len(line); i++ {
		if line[i] == 0x20 || line[i] == 0x00 {
			continue
		} else {
			start = i
			break
		}
	}

	//right trim point
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == 0x20 || line[i] == 0x00 {
			continue
		} else {
			stop = i + 1
			break
		}
	}
	return start, stop
}

// sgSesToBytes takes the []byte output from running "sg_ses --hex"
// drops all whitespace, and attempts to decode the hex charecters
// into to their actual values.
func sgSesToBytes(src []byte) ([]byte, int, error) {

	dropWhiteSpace := func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}

	src = bytes.Map(dropWhiteSpace, src)
	dst := make([]byte, hex.DecodedLen(len(src)))
	n, err := hex.Decode(dst, src)

	return dst, n, err
}
