//go:build linux && cgo && !agent

package cluster

// The code below was generated by lxd-generate - DO NOT EDIT!

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/lxc/lxd/lxd/db/query"
	"github.com/lxc/lxd/shared/api"
)

var _ = api.ServerEnvironment{}

const configObjects = `SELECT %s_config.id, %s_config.%s_id, %s_config.key, %s_config.value
  FROM %s_config
  ORDER BY %s_config.id`

const configCreate = `INSERT INTO %s_config (%s_id, key, value)
  VALUES (?, ?, ?)`

const configDelete = `DELETE FROM %s_config WHERE %s_id = ?`

// GetConfig returns all available config.
// generator: config GetMany
func GetConfig(ctx context.Context, tx *sql.Tx, parent string) (map[int]map[string]string, error) {
	var err error

	// Result slice.
	objects := make([]Config, 0)

	configObjectsLocal := strings.Replace(configObjects, "%s_id", fmt.Sprintf("%s_id", parent), -1)
	fillParent := make([]any, strings.Count(configObjectsLocal, "%s"))
	for i := range fillParent {
		fillParent[i] = strings.Replace(parent, "_", "s_", -1) + "s"
	}

	sqlStmt, err := prepare(tx, fmt.Sprintf(configObjectsLocal, fillParent...))
	if err != nil {
		return nil, err
	}

	args := []any{}

	// Dest function for scanning a row.
	dest := func(i int) []any {
		objects = append(objects, Config{})
		return []any{
			&objects[i].ID,
			&objects[i].ReferenceID,
			&objects[i].Key,
			&objects[i].Value,
		}
	}

	// Select.
	err = query.SelectObjects(sqlStmt, dest, args...)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch from \"config\" table: %w", err)
	}

	resultMap := map[int]map[string]string{}
	for _, object := range objects {
		if _, ok := resultMap[object.ReferenceID]; !ok {
			resultMap[object.ReferenceID] = map[string]string{}
		}
		resultMap[object.ReferenceID][object.Key] = object.Value
	}

	return resultMap, nil
}

// CreateConfig adds a new config to the database.
// generator: config Create
func CreateConfig(ctx context.Context, tx *sql.Tx, parent string, object Config) error {
	// An empty value means we are unsetting this key, so just return.
	if object.Value == "" {
		return nil
	}

	configCreateLocal := strings.Replace(configCreate, "%s_id", fmt.Sprintf("%s_id", parent), -1)
	fillParent := make([]any, strings.Count(configCreateLocal, "%s"))
	for i := range fillParent {
		fillParent[i] = strings.Replace(parent, "_", "s_", -1) + "s"
	}

	stmt, err := prepare(tx, fmt.Sprintf(configCreateLocal, fillParent...))
	if err != nil {
		return err
	}

	_, err = stmt.Exec(object.ReferenceID, object.Key, object.Value)
	if err != nil {
		return fmt.Errorf("Insert failed for \"%s_config\" table: %w", parent, err)
	}

	return nil
}

// UpdateConfig updates the config matching the given key parameters.
// generator: config Update
func UpdateConfig(ctx context.Context, tx *sql.Tx, parent string, referenceID int, config map[string]string) error {
	// Delete current entry.
	err := DeleteConfig(ctx, tx, parent, referenceID)
	if err != nil {
		return err
	}

	// Insert new entries.
	for key, value := range config {
		object := Config{
			ReferenceID: referenceID,
			Key:         key,
			Value:       value,
		}

		err = CreateConfig(ctx, tx, parent, object)
	}
	if err != nil {
		return err
	}

	return nil
}

// DeleteConfig deletes the config matching the given key parameters.
// generator: config DeleteMany
func DeleteConfig(ctx context.Context, tx *sql.Tx, parent string, referenceID int) error {
	configDeleteLocal := strings.Replace(configDelete, "%s_id", fmt.Sprintf("%s_id", parent), -1)
	fillParent := make([]any, strings.Count(configDeleteLocal, "%s"))
	for i := range fillParent {
		fillParent[i] = strings.Replace(parent, "_", "s_", -1) + "s"
	}

	stmt, err := prepare(tx, fmt.Sprintf(configDeleteLocal, fillParent...))
	if err != nil {
		return err
	}

	result, err := stmt.Exec(referenceID)
	if err != nil {
		return fmt.Errorf("Delete entry for \"%s_config\" failed: %w", parent, err)
	}

	_, err = result.RowsAffected()
	if err != nil {
		return fmt.Errorf("Fetch affected rows: %w", err)
	}

	return nil
}
