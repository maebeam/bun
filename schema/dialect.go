package schema

import (
	"database/sql"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/uptrace/bun/dialect"
	"github.com/uptrace/bun/dialect/feature"
)

type Dialect interface {
	Init(db *sql.DB)

	Name() dialect.Name
	Features() feature.Feature

	Tables() *Tables
	OnTable(table *Table)

	IdentQuote() byte

	AppendTime(b []byte, tm time.Time) []byte
	Append(fmter Formatter, b []byte, v interface{}) []byte
	Appender(typ reflect.Type) AppenderFunc

	FieldAppender(field *Field) AppenderFunc
	Scanner(typ reflect.Type) ScannerFunc
}

//------------------------------------------------------------------------------

type nopDialect struct {
	tables   *Tables
	features feature.Feature

	appenderMap sync.Map
	scannerMap  sync.Map
}

func newNopDialect() *nopDialect {
	d := new(nopDialect)
	d.tables = NewTables(d)
	d.features = feature.Returning
	return d
}

func (d *nopDialect) Init(*sql.DB) {}

func (d *nopDialect) Name() dialect.Name {
	return dialect.Invalid
}

func (d *nopDialect) Features() feature.Feature {
	return d.features
}

func (d *nopDialect) Tables() *Tables {
	return d.tables
}

func (d *nopDialect) OnField(field *Field) {}

func (d *nopDialect) OnTable(table *Table) {}

func (d *nopDialect) IdentQuote() byte {
	return '"'
}

func (d *nopDialect) AppendTime(b []byte, tm time.Time) []byte {
	b = append(b, '\'')
	b = tm.UTC().AppendFormat(b, "2006-01-02 15:04:05.999999-07:00")
	b = append(b, '\'')
	return b
}

func (d *nopDialect) Append(fmter Formatter, b []byte, v interface{}) []byte {
	switch v := v.(type) {
	case nil:
		return dialect.AppendNull(b)
	case bool:
		return dialect.AppendBool(b, v)
	case int:
		return strconv.AppendInt(b, int64(v), 10)
	case int32:
		return strconv.AppendInt(b, int64(v), 10)
	case int64:
		return strconv.AppendInt(b, v, 10)
	case uint:
		return strconv.AppendUint(b, uint64(v), 10)
	case uint32:
		return strconv.AppendUint(b, uint64(v), 10)
	case uint64:
		return strconv.AppendUint(b, v, 10)
	case float32:
		return dialect.AppendFloat32(b, v)
	case float64:
		return dialect.AppendFloat64(b, v)
	case string:
		return dialect.AppendString(b, v)
	case time.Time:
		return d.AppendTime(b, v)
	case []byte:
		return dialect.AppendBytes(b, v)
	case QueryAppender:
		return AppendQueryAppender(fmter, b, v)
	default:
		vv := reflect.ValueOf(v)
		if vv.Kind() == reflect.Ptr && vv.IsNil() {
			return dialect.AppendNull(b)
		}
		appender := Appender(vv.Type(), nil)
		return appender(fmter, b, vv)
	}
}

func (d *nopDialect) Appender(typ reflect.Type) AppenderFunc {
	if v, ok := d.appenderMap.Load(typ); ok {
		return v.(AppenderFunc)
	}

	fn := Appender(typ, nil)

	if v, ok := d.appenderMap.LoadOrStore(typ, fn); ok {
		return v.(AppenderFunc)
	}
	return fn
}

func (d *nopDialect) FieldAppender(field *Field) AppenderFunc {
	return FieldAppender(d, field)
}

func (d *nopDialect) Scanner(typ reflect.Type) ScannerFunc {
	if v, ok := d.scannerMap.Load(typ); ok {
		return v.(ScannerFunc)
	}

	fn := Scanner(typ)

	if v, ok := d.scannerMap.LoadOrStore(typ, fn); ok {
		return v.(ScannerFunc)
	}
	return fn
}
