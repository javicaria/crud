package crud

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

/*
Update syncs a tagged object with an existing record in the database.

The metadata contained in the crud tags don't include the table name or
the name of the SQL primary ID, so they have to be passed in manually.
If the object passed in as arg does not have a primary key set (or the
value is 0), an error is returned.
*/
func Update(db DbIsh, table, sqlIdFieldName string, arg interface{}) error {
	val := indirectV(reflect.ValueOf(arg))
	ty := val.Type()

	fieldMap, er := sqlToGoFields(ty)
	if er != nil {
		return er
	}

	sqlFields := make([]string, len(fieldMap))[:0]
	newValues := make([]interface{}, len(fieldMap))[:0]
	var id int64 = 0

	for sqlName, meta := range fieldMap {
		if meta.ReadOnly {
			continue
		}

		goName := meta.GoName

		if sqlName == sqlIdFieldName {
			id = val.FieldByName(goName).Int()

		} else {
			fieldVal := val.FieldByName(goName).Interface()

			if timeVal, ok := fieldVal.(time.Time); ok && meta.Unix {
				fieldVal = timeVal.Unix()

			} else if timeVal, ok := fieldVal.(*time.Time); ok && meta.Unix && timeVal != nil {
				fieldVal = timeVal.Unix()
			}

			sqlFields = append(sqlFields, fmt.Sprintf("%s = ?", sqlName))
			newValues = append(newValues, fieldVal)
		}
	}

	if id == 0 {
		return fmt.Errorf("%s is 0 or not set, cannot update", sqlIdFieldName)
	}

	newValues = append(newValues, id)

	q := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ?", table, strings.Join(sqlFields, ", "), sqlIdFieldName)
	_, er = db.Exec(q, newValues...)
	return er
}

/*
Insert creates a new record in the datastore.
*/
func Insert(db DbIsh, table, sqlIdFieldName string, arg interface{}) (int64, error) {
	val := indirectV(reflect.ValueOf(arg))
	ty := val.Type()

	fieldMap, er := sqlToGoFields(ty)
	if er != nil {
		return 0, er
	}

	sqlFields := make([]string, len(fieldMap))[:0]
	newValues := make([]interface{}, len(fieldMap))[:0]
	placeholders := make([]string, len(fieldMap))[:0]
	placeholderN := 1

	for sqlName, meta := range fieldMap {
		if meta.ReadOnly {
			continue
		}

		if sqlName == sqlIdFieldName {
			continue
		}

		goName := meta.GoName
		fieldVal := val.FieldByName(goName).Interface()

		if timeVal, ok := fieldVal.(time.Time); ok && meta.Unix {
			fieldVal = timeVal.Unix()

		} else if timeVal, ok := fieldVal.(*time.Time); ok && meta.Unix && timeVal != nil {
			fieldVal = timeVal.Unix()
		}

		sqlFields = append(sqlFields, sqlName)
		newValues = append(newValues, fieldVal)
		placeholders = append(placeholders, PlaceholderFormat(placeholderN))
		placeholderN++
	}

	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(sqlFields, ", "), strings.Join(placeholders, ", "))

	res, er := db.Exec(q, newValues...)
	if er != nil {
		fmt.Print(q)
		return 0, er
	}

	id, err := res.LastInsertId()
	if err != nil && NoLastInsertId {
		return 0, nil
	}

	return id, nil
}

/*
Upsert creates a new record in the datastore.
*/
func Upsert(db DbIsh, table, sqlIdFieldName string, arg interface{}) (int64, error) {
	if !EnableUpsert {
		return Insert(db, table, sqlIdFieldName, arg)
	}

	val := indirectV(reflect.ValueOf(arg))
	ty := val.Type()

	fieldMap, er := sqlToGoFields(ty)
	if er != nil {
		return 0, er
	}

	sqlFields := make([]string, len(fieldMap))[:0]
	newValues := make([]interface{}, len(fieldMap))[:0]
	placeholders := make([]string, len(fieldMap))[:0]
	placeholderN := 1

	sqlUpsertFields := make([]string, len(fieldMap))[:0]

	for sqlName, meta := range fieldMap {
		if meta.ReadOnly {
			continue
		}

		if sqlName == sqlIdFieldName {
			continue
		}

		goName := meta.GoName
		fieldVal := val.FieldByName(goName).Interface()

		if timeVal, ok := fieldVal.(time.Time); ok && meta.Unix {
			fieldVal = timeVal.Unix()

		} else if timeVal, ok := fieldVal.(*time.Time); ok && meta.Unix && timeVal != nil {
			fieldVal = timeVal.Unix()
		}

		sqlFields = append(sqlFields, sqlName)
		newValues = append(newValues, fieldVal)
		placeholders = append(placeholders, PlaceholderFormat(placeholderN))
		placeholderN++

		sqlUpsertFields = append(sqlUpsertFields, fmt.Sprintf("%s = ?", sqlName))
	}

	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s", table, strings.Join(sqlFields, ", "), strings.Join(placeholders, ", "), strings.Join(sqlUpsertFields, ", "))

	newValues = append(newValues, newValues...)

	res, er := db.Exec(q, newValues...)
	if er != nil {
		return 0, er
	}

	id, err := res.LastInsertId()
	if err != nil && NoLastInsertId {
		return 0, nil
	}

	return id, err
}
