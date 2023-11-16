package data

import (
	"fmt"
	"reflect"
	"strconv"
)

type SpawnRecord struct {
	Build      int
	Race       uint8
	Class      uint8
	MapID      int
	ZoneID     int
	X, Y, Z, O float32
}

func Unmarshal(record []string, v interface{}) error {
	s := reflect.ValueOf(v).Elem()
	if s.NumField() != len(record) {
		return fmt.Errorf("field mismatch: %d, %d", s.NumField(), len(record))
	}

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		switch f.Kind() {
		case reflect.String:
			f.SetString(record[i])

		case reflect.Int:
			ival, err := strconv.ParseInt(record[i], 10, 0)
			if err != nil {
				return err
			}
			f.SetInt(ival)

		case reflect.Uint8:
			ival, err := strconv.ParseInt(record[i], 10, 0)
			if err != nil {
				return err
			}
			f.SetUint(uint64(ival))

		case reflect.Float32:
			ival, err := strconv.ParseFloat(record[i], 32)
			if err != nil {
				return err
			}
			f.SetFloat(ival)

		default:
			return fmt.Errorf("unsupported type: %s", f.Type().String())
		}
	}

	return nil
}
