package sastopo

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
