package protocol

const (
	// record type
	RecordTypeA     uint16 = 1
	RecordTypeNS    uint16 = 2
	RecordTypeCNAME uint16 = 5
	RecordTypeAAAA  uint16 = 28

	// record class
	RecordClassIN uint16 = 1

	// other
	offsetFlagExcess uint16 = 0b11000000 << 8
)
