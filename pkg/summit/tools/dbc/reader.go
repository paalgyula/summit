package dbc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
)

// DataHeader is the header of a DBC file with the following fields:
// Magic: always 'WDBC'
// RecordCount: records per file
// FieldCount: fields per record.
type DataHeader struct {
	Magic [4]byte // always 'WDBC'
	// records per file
	RecordCount uint32
	// fields per record. The field disze is always 4bytes long (uint32)
	FieldCount uint32
	// RecordSize is the size of a record in bytes
	RecordSize uint32
	// StringBlockSize size of the string block at the end of file in bytes
	StringBlockSize uint32
}

type Reader[C any] struct {
	r      io.Reader
	Header DataHeader

	current int

	Records []C
}

// NewReader creates a new Reader instance for the given io.Reader to read DBC files.
//
//	r: the io.Reader to read DBC files from.
//	(*Reader[C], error): a pointer to a Reader instance and a possible error that might occur.
func NewReader[C any](r io.Reader) (*Reader[C], error) {
	dbcReader := &Reader[C]{
		r:      r,
		Header: DataHeader{},
	}

	err := binary.Read(r, binary.LittleEndian, &dbcReader.Header)
	if err != nil {
		return nil, fmt.Errorf("cannot read DBC header: %w", err)
	}

	fmt.Printf("Header: records: %d, record size: %d, string block size: %d\n",
		dbcReader.Header.RecordCount,
		dbcReader.Header.RecordSize,
		dbcReader.Header.StringBlockSize,
	)

	return dbcReader, nil
}

// ReadAll reads all records from a Reader and stores them in its Records
// field, as well as parses the strings from the string block. Returns an error
// if the expected number of bytes for the string block is not present.
// Returns nil if no errors occurred.
func (dr *Reader[C]) ReadAll() error {
	row := make([]byte, dr.Header.RecordSize)

	dr.Records = make([]C, dr.Header.RecordCount)

	for ; dr.current < int(dr.Header.RecordCount); dr.current++ {
		_, _ = dr.r.Read(row)
		// data, err := dr.ParseRow(row)
		// fmt.Printf(">> %s", hex.Dump(row))
		var data C

		parseByteArray(row, &data)
		dr.Records[dr.current] = data
	}

	strings, _ := io.ReadAll(dr.r)
	if len(strings) != int(dr.Header.StringBlockSize) {
		return fmt.Errorf("expected %d bytes, got %d", dr.Header.StringBlockSize, len(strings))
	}

	// fmt.Printf(">> %s", hex.Dump(strings))

	dr.parseStrings(strings)

	// fmt.Printf("%+v", dr.Records)

	return nil
}

// parseStrings parses a byte slice containing strings and populates the Records
// field of the Reader with the parsed data.
//
// strings: a byte slice containing strings.
// error: an error is returned if there was an error parsing the data.
func (dr *Reader[C]) parseStrings(strings []byte) error {
	r := bytes.NewReader(strings)

	for i := 0; i < len(dr.Records); i++ {
		v := reflect.ValueOf(&dr.Records[i]).Elem()

		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)

			// LocalizedString
			if field.Type() == reflect.TypeOf(wotlk.LocalizedString{}) {
				val, _ := reflect.Value(field).Interface().(wotlk.LocalizedString)
				for _, l := range val.Locales {
					if l != nil {
						r.Seek(int64(l.Location), io.SeekStart)
						s := readCstring(r)

						l.Value = s
					}
				}
			}

			// String reference
			if field.Type() == reflect.TypeOf((*wotlk.StringRef)(nil)) {
				sr, _ := reflect.Value(field).Interface().(*wotlk.StringRef)
				r.Seek(int64(sr.Location), io.SeekStart)
				s := readCstring(r)

				sr.Value = s
			}
		}
	}

	return nil
}

// readCstring reads bytes from an io.Reader until a null byte is found and
// returns the resulting string. It takes a single parameter, an io.Reader, and
// returns a string.
func readCstring(r io.Reader) string {
	s := bytes.NewBufferString("")

	for {
		bb := make([]byte, 1)

		if _, err := r.Read(bb); err != nil {
			return s.String()
		}

		if bb[0] == '\x00' {
			break
		}

		s.Write(bb)
	}

	return s.String()
}

//nolint:funlen
func parseByteArray(data []byte, obj interface{}) error {
	v := reflect.ValueOf(obj).Elem()

	offset := 0

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		dbcTag := v.Type().Field(i).Tag.Get("dbc")

		if dbcTag == "" {
			continue
		}

		size := 0
		b := 0

		fmt.Sscanf(dbcTag, "offset=%d", &offset)
		fmt.Sscanf(dbcTag, "offset=%d,byte=%d", &offset, &b)
		fmt.Sscanf(dbcTag, "offset=%d,len=%d", &offset, &size)

		offset *= 4 + b // Bytes to columns + byte
		size *= 4

		var value any

		switch field.Kind() {
		case reflect.Int8:
			value = int8(data[offset])
		case reflect.Uint8:
			value = data[offset]
		case reflect.String:
			value = string(data[offset : offset+size])
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.Uint8 {
				value = data[offset : offset+size]
			} else if field.Type().Elem().Kind() == reflect.Uint32 {
				value := make([]uint32, size/4)
				br := bytes.NewReader(data[offset : offset+size])
				err := binary.Read(br, binary.LittleEndian, &value)
				if err != nil {
					return err
				}

				field.Set(reflect.ValueOf(value))

				continue
				// fmt.Printf("data type not supported: %v\n", field.Type().Elem().Kind())
			} else {
				fmt.Printf("data type not supported: %v\n", field.Type().Elem().Kind())
			}
		case reflect.Int16:
			value = int16(binary.LittleEndian.Uint16(data[offset:]))
		case reflect.Uint16:
			value = binary.LittleEndian.Uint16(data[offset:])
		case reflect.Int32:
			value = int32(binary.LittleEndian.Uint32(data[offset:]))
		case reflect.Uint32:
			value = binary.LittleEndian.Uint32(data[offset:])
		case reflect.Int64:
			value = int64(binary.LittleEndian.Uint64(data[offset:]))
		case reflect.Uint64:
			value = binary.LittleEndian.Uint64(data[offset:])
		case reflect.Pointer:
			value = parsePointer(field, data, offset)
		default:
			value = parseStruct(field, data, offset)
			if value == nil {
				fmt.Printf("unsupported type %+v %v\n", field.Kind(), field.Type())
			}
		}

		if value != nil {
			field.Set(reflect.ValueOf(value))
		}
	}

	return nil
}

func parsePointer(field reflect.Value, data []byte, offset int) any {
	// field := reflect.TypeOf((*wotlk.StringRef)(nil)).Elem()
	fieldType := field.Type()

	switch fieldType {
	case reflect.TypeOf((*wotlk.StringRef)(nil)):
		var location uint32 = binary.LittleEndian.Uint32(data[offset:])
		// binary.Read(br, binary.LittleEndian, &location)

		return &wotlk.StringRef{
			Location: location,
		}
	default:
		return nil
	}
}

func parseStruct(field reflect.Value, data []byte, offset int) any {
	var value any
	if field.Type() == reflect.TypeOf(wotlk.LocalizedString{}) {
		value = wotlk.CreatesLocalizedString(data[offset:])
	} else if field.Type() == reflect.TypeOf(wotlk.StringRef{}) {
		panic("StringRef should be a pointer type")
	}

	return value
}
