package global

/*
   Creation Time: 2021 - Aug - 05
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2020
*/

type ErrorCode int

const (
	ErrUnknown     ErrorCode = 0x00
	ErrAccess      ErrorCode = 0x01
	ErrUnavailable ErrorCode = 0x02
	ErrInvalid     ErrorCode = 0x03
	ErrIncomplete  ErrorCode = 0x04
	ErrDuplicate   ErrorCode = 0x05
	ErrLimit       ErrorCode = 0x06
	ErrTimeout     ErrorCode = 0x07
	ErrSession     ErrorCode = 0x08
)
