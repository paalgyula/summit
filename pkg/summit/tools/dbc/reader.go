package dbc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/paalgyula/summit/pkg/summit/tools/dbc/wotlk"
	"github.com/rs/zerolog/log"
)

var (
	ErrInvalidMagic     = errors.New("invalid DBC magic, expected WDBC")
	ErrShortStringBlock = errors.New("string block size mismatch")
)

type DataHeader struct {
	Magic           [4]byte
	RecordCount     uint32
	FieldCount      uint32
	RecordSize      uint32
	StringBlockSize uint32
}

type Reader[C any] struct {
	r       io.Reader
	Header  DataHeader
	Records []C
}

func NewReader[C any](r io.Reader) (*Reader[C], error) {
	//nolint:exhaustruct
	dbcReader := &Reader[C]{
		r: r,
	}

	if err := binary.Read(r, binary.LittleEndian, &dbcReader.Header); err != nil {
		return nil, fmt.Errorf("cannot read DBC header: %w", err)
	}

	if string(dbcReader.Header.Magic[:]) != "WDBC" {
		return nil, fmt.Errorf("%w: got %q", ErrInvalidMagic, dbcReader.Header.Magic)
	}

	log.Debug().Msgf("Header: records: %d, record size: %d, string block size: %d",
		dbcReader.Header.RecordCount,
		dbcReader.Header.RecordSize,
		dbcReader.Header.StringBlockSize,
	)

	return dbcReader, nil
}

type fieldInfo struct {
	idx      int
	byteOff  int
	byteSize int
	kind     reflect.Kind
	elemKind reflect.Kind
}

func (dr *Reader[C]) ReadAll() error {
	fields := cachedFields[C]()

	row := make([]byte, dr.Header.RecordSize)
	dr.Records = make([]C, dr.Header.RecordCount)

	for i := range dr.Records {
		if _, err := io.ReadFull(dr.r, row); err != nil {
			return fmt.Errorf("reading record %d: %w", i, err)
		}

		dr.Records[i] = parseRow[C](row, fields)
	}

	strings, err := io.ReadAll(dr.r)
	if err != nil {
		return fmt.Errorf("reading string block: %w", err)
	}

	if len(strings) != int(dr.Header.StringBlockSize) {
		return fmt.Errorf("%w: expected %d bytes, got %d",
			ErrShortStringBlock, dr.Header.StringBlockSize, len(strings))
	}

	resolveStrings(dr.Records, fields, strings)

	return nil
}

//nolint:cyclop,gosec,ireturn
func parseRow[C any](row []byte, fields []fieldInfo) C {
	v := reflect.ValueOf(new(C)).Elem()

	for _, f := range fields {
		off := f.byteOff

		//nolint:exhaustive
		switch f.kind {
		case reflect.Uint8:
			v.Field(f.idx).SetUint(uint64(row[off]))
		case reflect.Int8:
			v.Field(f.idx).SetInt(int64(int8(row[off])))
		case reflect.Uint16:
			v.Field(f.idx).SetUint(uint64(binary.LittleEndian.Uint16(row[off:])))
		case reflect.Int16:
			v.Field(f.idx).SetInt(int64(int16(binary.LittleEndian.Uint16(row[off:]))))
		case reflect.Uint32:
			v.Field(f.idx).SetUint(uint64(binary.LittleEndian.Uint32(row[off:])))
		case reflect.Int32:
			v.Field(f.idx).SetInt(int64(int32(binary.LittleEndian.Uint32(row[off:]))))
		case reflect.Uint64:
			v.Field(f.idx).SetUint(binary.LittleEndian.Uint64(row[off:]))
		case reflect.Int64:
			v.Field(f.idx).SetInt(int64(binary.LittleEndian.Uint64(row[off:])))
		case reflect.String:
			v.Field(f.idx).SetString(string(row[off : off+f.byteSize]))
		case reflect.Slice:
			if f.elemKind == reflect.Uint8 {
				dst := make([]byte, f.byteSize)
				copy(dst, row[off:off+f.byteSize])
				v.Field(f.idx).SetBytes(dst)
			} else if f.elemKind == reflect.Uint32 {
				count := f.byteSize / fieldSize
				dst := make([]uint32, count)

				for j := range dst {
					dst[j] = binary.LittleEndian.Uint32(row[off+j*fieldSize:])
				}

				v.Field(f.idx).Set(reflect.ValueOf(dst))
			}
		case reflect.Pointer:
			location := binary.LittleEndian.Uint32(row[off:])
			//nolint:exhaustruct
			v.Field(f.idx).Set(reflect.ValueOf(&wotlk.StringRef{Location: location}))
		case reflect.Struct:
			ls := wotlk.CreatesLocalizedString(row[off : off+f.byteSize])
			v.Field(f.idx).Set(reflect.ValueOf(ls))
		}
	}

	//nolint:forcetypeassert
	return v.Interface().(C)
}

//nolint:exhaustive
func resolveStrings[C any](records []C, fields []fieldInfo, strings []byte) {
	for i := range records {
		v := reflect.ValueOf(&records[i]).Elem()

		for _, f := range fields {
			//nolint:exhaustive
			switch f.kind {
			case reflect.Pointer:
				//nolint:forcetypeassert
				sr := v.Field(f.idx).Interface().(*wotlk.StringRef)
				if sr != nil {
					sr.Value = readCstringAt(strings, int(sr.Location))
				}
			case reflect.Struct:
				//nolint:forcetypeassert
				ls := v.Field(f.idx).Interface().(wotlk.LocalizedString)
				for _, l := range ls.Locales {
					if l != nil {
						l.Value = readCstringAt(strings, int(l.Location))
					}
				}
			}
		}
	}
}

func readCstringAt(data []byte, off int) string {
	end := off
	for end < len(data) && data[end] != 0 {
		end++
	}

	return string(data[off:end])
}

const fieldSize = 4

func cachedFields[C any]() []fieldInfo {
	var zero C
	t := reflect.TypeOf(zero)

	n := t.NumField()
	fields := make([]fieldInfo, 0, n)

	for i := 0; i < n; i++ {
		f := t.Field(i)
		tag := f.Tag.Get("dbc")

		if tag == "" {
			continue
		}

		colOff, byteOff, count := parseTag(tag)

		//nolint:exhaustruct
		fi := fieldInfo{
			idx:     i,
			byteOff: colOff*fieldSize + byteOff,
			kind:    f.Type.Kind(),
		}

		//nolint:exhaustive
		switch f.Type.Kind() {
		case reflect.String:
			fi.byteSize = count * fieldSize
		case reflect.Slice:
			fi.elemKind = f.Type.Elem().Kind()
			fi.byteSize = count * fieldSize
		case reflect.Struct:
			fi.byteSize = numLocales * fieldSize
		case reflect.Pointer:
			fi.byteSize = fieldSize
		}

		fields = append(fields, fi)
	}

	return fields
}

const numLocales = 16

const tagParts = 2

//nolint:nonamedreturns
func parseTag(tag string) (offset int, byteOff int, size int) {
	for _, part := range strings.Split(tag, ",") {
		kv := strings.SplitN(part, "=", tagParts)
		if len(kv) != tagParts {
			continue
		}

		val, err := strconv.Atoi(kv[1])
		if err != nil {
			continue
		}

		switch kv[0] {
		case "offset":
			offset = val
		case "byte":
			byteOff = val
		case "len":
			size = val
		}
	}

	return
}
