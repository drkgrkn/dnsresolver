package main

type Flags uint16

func (f Flags) RecursionDesired(recursionDesired bool) Flags {
	mask := Flags(0x0A00)
	if recursionDesired {
		f |= mask
	} else {
		mask = ^mask
		f &= mask
	}

	return f
}
