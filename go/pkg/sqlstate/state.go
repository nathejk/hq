package sqlstate

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"nathejk.dk/pkg/streamtable"
)

type state struct {
	db *sql.DB
}

func New(dsn string) *state {
	db, err := sql.Open("mysql", dsn)

	if err != nil {
		log.Fatal(err)
		return nil
	}

	return &state{db: db}
}
func (s *state) Close() error {
	return s.db.Close()
}

func createSql(name string, fs []reflect.StructField) string {
	columns := []string{}
	for _, field := range fs {
		name := field.Name
		typ := ""
		null := "NOT NULL"
		fieldType := field.Type.String()
		if fieldType[0] == '*' {
			fieldType = fieldType[1:]
			null = ""
		}
		switch fieldType {
		case "int":
			typ = "INTEGER"
		default:
			typ = "TEXT"
		}

		columns = append(columns, strings.TrimRight(fmt.Sprintf("  `%s` %s %s", name, typ, null), " "))
	}
	columnSql := strings.Join(columns, ",\n")
	return fmt.Sprintf("CREATE TABLE `%s` (\n%s\n)", name, columnSql)
}

func fields(t reflect.Type) []reflect.StructField {
	fs := make([]reflect.StructField, 0)

	// Return if not struct or pointer to struct.
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return fs
	}
	//return reflect.VisibleFields(t) // go 1.17

	// Iterate through fields collecting names in map.
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		// Recurse into anonymous fields (embedded structs).
		if f.Anonymous {
			fs = append(fs, fields(f.Type)...)
		} else {
			fs = append(fs, f)
		}
	}
	return fs
}

func (s *state) Init(entity streamtable.Entity) error {
	t := reflect.TypeOf(entity)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return fmt.Errorf("Not a struct or pointer to struct")
	}
	ss := strings.Split(t.String(), ".")
	table := ss[len(ss)-1]
	sql := createSql(table, fields(t))
	log.Println(sql)
	if err := s.Query(fmt.Sprintf("DROP TABLE IF EXISTS `%s`", table)); err != nil {
		return err
	}
	return s.Query(sql)
}

func (s *state) Write(entity streamtable.Entity) error {
	//spew.Dump(entity)
	return nil
}

func (s *state) Query(sql string) error {
	res, err := s.db.Query(sql)
	if err != nil {
		return err
	}
	for res.Next() {
	}
	return nil
}
