package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"toolBox/modules/test-sheet/pkg/model"

	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultDatabaseDirectory = "BDD"
	defaultDatabaseFile      = "test-sheet.db"
)

type SQLiteRepository struct {
	db *sql.DB
}

func Open(path string) (*SQLiteRepository, error) {
	if path == "" {
		path = DefaultPath()
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("create database directory %s: %w", directory, err)
	}
	exists, err := databaseFileExists(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open database %s: %w", path, err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create or open database %s: %w", path, err)
	}
	if !exists {
		if _, err := os.Stat(path); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("database file was not created %s: %w", path, err)
		}
	}
	repo := &SQLiteRepository{db: db}
	if err := repo.Migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate database %s: %w", path, err)
	}
	return repo, nil
}

func DefaultPath() string {
	return filepath.Join(executableDirectory(), defaultDatabaseDirectory, defaultDatabaseFile)
}

func executableDirectory() string {
	executable, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(executable)
}

func databaseFileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("check database file %s: %w", path, err)
	}
	return true, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

func (r *SQLiteRepository) DB() *sql.DB {
	return r.db
}

func (r *SQLiteRepository) Migrate() error {
	if err := r.renameLegacySheetsTable(); err != nil {
		return err
	}
	if _, err := r.db.Exec(migrationSQL); err != nil {
		return err
	}
	for _, column := range []string{"config", "command", "notes"} {
		if err := r.ensureTextColumn("test_sheets", column); err != nil {
			return err
		}
		if err := r.ensureTextColumn("test_run_sheets", column); err != nil {
			return err
		}
	}
	if err := r.ensureTextColumn("test_runs", "group_name"); err != nil {
		return err
	}
	if err := r.ensureNullableDateTimeColumn("test_plans", "deleted_at"); err != nil {
		return err
	}
	if err := r.ensureIntegerColumn("test_run_evidences", "size_bytes"); err != nil {
		return err
	}
	if err := r.ensureNullableIntegerColumn("test_sheets", "group_id"); err != nil {
		return err
	}
	if err := r.ensureNullableIntegerColumn("test_runs", "group_id"); err != nil {
		return err
	}
	if err := r.ensureNullableIntegerColumn("test_run_sheets", "run_group_id"); err != nil {
		return err
	}
	if _, err := r.db.Exec(documentMigrationSQL); err != nil {
		return err
	}
	if err := r.migrateDefaultGroups(); err != nil {
		return err
	}
	if err := r.migrateDefaultRunGroups(); err != nil {
		return err
	}
	return r.migrateLegacySteps()
}

func (r *SQLiteRepository) ensureTextColumn(table, column string) error {
	exists, err := r.columnExists(table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = r.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s TEXT NOT NULL DEFAULT ''", table, column))
	return err
}

func (r *SQLiteRepository) ensureNullableDateTimeColumn(table, column string) error {
	exists, err := r.columnExists(table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = r.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s DATETIME", table, column))
	return err
}

func (r *SQLiteRepository) ensureIntegerColumn(table, column string) error {
	exists, err := r.columnExists(table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = r.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s INTEGER NOT NULL DEFAULT 0", table, column))
	return err
}

func (r *SQLiteRepository) ensureNullableIntegerColumn(table, column string) error {
	exists, err := r.columnExists(table, column)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = r.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s INTEGER", table, column))
	return err
}

func (r *SQLiteRepository) migrateDefaultGroups() error {
	_, err := r.db.Exec(`
INSERT INTO test_plan_groups (plan_id, name, description, execution_order, created_at, updated_at)
SELECT p.id, 'Sous-plan principal', '', 1, p.created_at, p.updated_at
FROM test_plans p
WHERE NOT EXISTS (SELECT 1 FROM test_plan_groups g WHERE g.plan_id = p.id);

UPDATE test_sheets
SET group_id = (
	SELECT g.id FROM test_plan_groups g
	WHERE g.plan_id = test_sheets.plan_id
	ORDER BY g.execution_order ASC, g.id ASC
	LIMIT 1
)
WHERE group_id IS NULL;

UPDATE test_runs
SET group_id = (
	SELECT g.id FROM test_plan_groups g
	WHERE g.plan_id = test_runs.plan_id
	ORDER BY g.execution_order ASC, g.id ASC
	LIMIT 1
)
WHERE group_id IS NULL;

UPDATE test_runs
SET group_name = COALESCE((SELECT g.name FROM test_plan_groups g WHERE g.id = test_runs.group_id), '')
WHERE TRIM(group_name) = '';
`)
	return err
}

func (r *SQLiteRepository) migrateDefaultRunGroups() error {
	_, err := r.db.Exec(`
INSERT INTO test_run_groups (run_id, source_group_id, name, description, execution_order, created_at)
SELECT r.id, r.group_id, COALESCE(NULLIF(r.group_name, ''), g.name, 'Sous-plan principal'), COALESCE(g.description, ''), COALESCE(g.execution_order, 1), r.started_at
FROM test_runs r
LEFT JOIN test_plan_groups g ON g.id = r.group_id
WHERE NOT EXISTS (SELECT 1 FROM test_run_groups rg WHERE rg.run_id = r.id);

UPDATE test_run_sheets
SET run_group_id = (
	SELECT rg.id FROM test_run_groups rg WHERE rg.run_id = test_run_sheets.run_id ORDER BY rg.execution_order ASC, rg.id ASC LIMIT 1
)
WHERE run_group_id IS NULL;
`)
	return err
}

func (r *SQLiteRepository) columnExists(table, column string) (bool, error) {
	rows, err := r.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func (r *SQLiteRepository) migrateLegacySteps() error {
	_, err := r.db.Exec(`
INSERT INTO test_sheet_steps (sheet_id, action, field, expected_result, execution_order, created_at, updated_at)
SELECT s.id, s.action, '', s.expected_result, 1, s.created_at, s.updated_at
FROM test_sheets s
WHERE (TRIM(s.action) <> '' OR TRIM(s.expected_result) <> '')
	AND NOT EXISTS (SELECT 1 FROM test_sheet_steps step WHERE step.sheet_id = s.id);

INSERT INTO test_run_steps (run_sheet_id, source_step_id, action, field, expected_result, execution_order, status, actual_result, comment, created_at, updated_at)
SELECT rs.id, NULL, rs.action, '', rs.expected_result, 1, rs.status, rs.actual_result, rs.comment, rs.created_at, rs.updated_at
FROM test_run_sheets rs
WHERE (TRIM(rs.action) <> '' OR TRIM(rs.expected_result) <> '')
	AND NOT EXISTS (SELECT 1 FROM test_run_steps step WHERE step.run_sheet_id = rs.id);
`)
	return err
}

func (r *SQLiteRepository) renameLegacySheetsTable() error {
	rows, err := r.db.Query(`PRAGMA table_info(test_sheets)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	hasTable := false
	hasPlanID := false
	for rows.Next() {
		hasTable = true
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue any
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == "plan_id" {
			hasPlanID = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if hasTable && !hasPlanID {
		_, err := r.db.Exec(fmt.Sprintf(`ALTER TABLE test_sheets RENAME TO test_sheets_legacy_%d`, time.Now().UTC().Unix()))
		return err
	}
	return nil
}

func (r *SQLiteRepository) CreatePlan(input model.PlanInput) (model.TestPlan, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO test_plans (name, description, mockup_settings, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		input.Name, input.Description, input.MockupSettings, now, now)
	if err != nil {
		return model.TestPlan{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.TestPlan{}, err
	}
	if _, err := r.CreateGroup(id, model.GroupInput{Name: "Sous-plan principal", ExecutionOrder: 1}); err != nil {
		return model.TestPlan{}, err
	}
	return r.GetPlan(id)
}

func (r *SQLiteRepository) ListPlans() ([]model.TestPlan, error) {
	rows, err := r.db.Query(`SELECT id, name, description, mockup_settings, created_at, updated_at, deleted_at FROM test_plans WHERE deleted_at IS NULL ORDER BY updated_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plans := []model.TestPlan{}
	for rows.Next() {
		plan, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func (r *SQLiteRepository) GetPlan(id int64) (model.TestPlan, error) {
	row := r.db.QueryRow(`SELECT id, name, description, mockup_settings, created_at, updated_at, deleted_at FROM test_plans WHERE id = ?`, id)
	return scanPlan(row)
}

func (r *SQLiteRepository) TouchPlan(id int64) error {
	res, err := r.db.Exec(`UPDATE test_plans SET updated_at = ? WHERE id = ?`, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLiteRepository) UpdatePlan(id int64, input model.PlanInput) (model.TestPlan, error) {
	res, err := r.db.Exec(`UPDATE test_plans SET name = ?, description = ?, mockup_settings = ?, updated_at = ? WHERE id = ?`,
		input.Name, input.Description, input.MockupSettings, time.Now().UTC(), id)
	if err != nil {
		return model.TestPlan{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestPlan{}, sql.ErrNoRows
	}
	return r.GetPlan(id)
}

func (r *SQLiteRepository) DeletePlan(id int64) error {
	now := time.Now().UTC()
	res, err := r.db.Exec(`UPDATE test_plans SET deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLiteRepository) PermanentDeletePlan(id int64) error {
	res, err := r.db.Exec(`DELETE FROM test_plans WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLiteRepository) RestorePlan(id int64) (model.TestPlan, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`UPDATE test_plans SET deleted_at = NULL, updated_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return model.TestPlan{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestPlan{}, sql.ErrNoRows
	}
	return r.GetPlan(id)
}

func (r *SQLiteRepository) CreateGroup(planID int64, input model.GroupInput) (model.TestGroup, error) {
	if input.ExecutionOrder == 0 {
		next, err := r.nextGroupOrder(planID)
		if err != nil {
			return model.TestGroup{}, err
		}
		input.ExecutionOrder = next
	}
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO test_plan_groups (plan_id, name, description, execution_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`, planID, input.Name, input.Description, input.ExecutionOrder, now, now)
	if err != nil {
		return model.TestGroup{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.TestGroup{}, err
	}
	return r.GetGroup(id)
}

func (r *SQLiteRepository) ListGroups(planID int64) ([]model.TestGroup, error) {
	rows, err := r.db.Query(`SELECT g.id, g.plan_id, g.name, g.description, g.execution_order, g.created_at, g.updated_at, COUNT(s.id) AS sheet_count
		FROM test_plan_groups g
		LEFT JOIN test_sheets s ON s.group_id = g.id
		WHERE g.plan_id = ?
		GROUP BY g.id
		ORDER BY g.execution_order ASC, g.id ASC`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := []model.TestGroup{}
	for rows.Next() {
		group, err := scanGroupWithSheetCount(rows)
		if err != nil {
			return nil, err
		}
		runs, err := r.ListGroupRuns(group.ID)
		if err != nil {
			return nil, err
		}
		group.RunCount = len(runs)
		if len(runs) > 0 {
			latest := runs[0]
			group.LatestRun = &latest
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (r *SQLiteRepository) GetGroup(id int64) (model.TestGroup, error) {
	row := r.db.QueryRow(`SELECT id, plan_id, name, description, execution_order, created_at, updated_at
		FROM test_plan_groups WHERE id = ?`, id)
	group, err := scanGroup(row)
	if err != nil {
		return model.TestGroup{}, err
	}
	group.Sheets, err = r.ListSheetsByGroup(id)
	if err != nil {
		return model.TestGroup{}, err
	}
	group.SheetCount = len(group.Sheets)
	runs, err := r.ListGroupRuns(id)
	if err != nil {
		return model.TestGroup{}, err
	}
	group.RunCount = len(runs)
	if len(runs) > 0 {
		latest := runs[0]
		group.LatestRun = &latest
	}
	return group, nil
}

func (r *SQLiteRepository) UpdateGroup(id int64, input model.GroupInput) (model.TestGroup, error) {
	res, err := r.db.Exec(`UPDATE test_plan_groups SET name = ?, description = ?, execution_order = ?, updated_at = ? WHERE id = ?`,
		input.Name, input.Description, input.ExecutionOrder, time.Now().UTC(), id)
	if err != nil {
		return model.TestGroup{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestGroup{}, sql.ErrNoRows
	}
	return r.GetGroup(id)
}

func (r *SQLiteRepository) TouchGroup(id int64) error {
	res, err := r.db.Exec(`UPDATE test_plan_groups SET updated_at = ? WHERE id = ?`, time.Now().UTC(), id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLiteRepository) DeleteGroup(id int64) error {
	res, err := r.db.Exec(`DELETE FROM test_plan_groups WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLiteRepository) ReorderGroups(planID int64, groupIDs []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer rollback(tx)
	for index, groupID := range groupIDs {
		res, err := tx.Exec(`UPDATE test_plan_groups SET execution_order = ?, updated_at = ? WHERE id = ? AND plan_id = ?`,
			index+1, time.Now().UTC(), groupID, planID)
		if err != nil {
			return err
		}
		if changed, _ := res.RowsAffected(); changed == 0 {
			return sql.ErrNoRows
		}
	}
	return tx.Commit()
}

func (r *SQLiteRepository) DefaultGroupID(planID int64) (int64, error) {
	var id int64
	err := r.db.QueryRow(`SELECT id FROM test_plan_groups WHERE plan_id = ? ORDER BY execution_order ASC, id ASC LIMIT 1`, planID).Scan(&id)
	return id, err
}

func (r *SQLiteRepository) CreateSheet(planID int64, input model.SheetInput) (model.TestSheet, error) {
	groupID, err := r.DefaultGroupID(planID)
	if err != nil {
		return model.TestSheet{}, err
	}
	return r.CreateSheetInGroup(groupID, input)
}

func (r *SQLiteRepository) CreateSheetInGroup(groupID int64, input model.SheetInput) (model.TestSheet, error) {
	group, err := r.GetGroup(groupID)
	if err != nil {
		return model.TestSheet{}, err
	}
	if input.ExecutionOrder == 0 {
		next, err := r.nextSheetOrder(groupID)
		if err != nil {
			return model.TestSheet{}, err
		}
		input.ExecutionOrder = next
	}
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO test_sheets
		(plan_id, group_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, mockup_settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		group.PlanID, groupID, input.Name, input.Description, input.Prerequisites, input.Config, input.Command, input.Notes, input.Action, input.ExpectedResult, input.ExecutionOrder, input.MockupSettings, now, now)
	if err != nil {
		return model.TestSheet{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.TestSheet{}, err
	}
	return r.GetSheet(id)
}

func (r *SQLiteRepository) ListSheets(planID int64) ([]model.TestSheet, error) {
	rows, err := r.db.Query(`SELECT id, plan_id, COALESCE(group_id, 0), name, description, prerequisites, config, command, notes, action, expected_result, execution_order, mockup_settings, created_at, updated_at
		FROM test_sheets WHERE plan_id = ? ORDER BY execution_order ASC, id ASC`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sheets, err := scanSheets(rows)
	if err != nil {
		return nil, err
	}
	for index := range sheets {
		steps, err := r.ListSteps(sheets[index].ID)
		if err != nil {
			return nil, err
		}
		sheets[index].Steps = steps
		documents, err := r.ListSheetDocuments(sheets[index].ID)
		if err != nil {
			return nil, err
		}
		sheets[index].Documents = documents
	}
	return sheets, nil
}

func (r *SQLiteRepository) ListSheetsByGroup(groupID int64) ([]model.TestSheet, error) {
	rows, err := r.db.Query(`SELECT id, plan_id, COALESCE(group_id, 0), name, description, prerequisites, config, command, notes, action, expected_result, execution_order, mockup_settings, created_at, updated_at
		FROM test_sheets WHERE group_id = ? ORDER BY execution_order ASC, id ASC`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sheets, err := scanSheets(rows)
	if err != nil {
		return nil, err
	}
	for index := range sheets {
		steps, err := r.ListSteps(sheets[index].ID)
		if err != nil {
			return nil, err
		}
		sheets[index].Steps = steps
		documents, err := r.ListSheetDocuments(sheets[index].ID)
		if err != nil {
			return nil, err
		}
		sheets[index].Documents = documents
	}
	return sheets, nil
}

func (r *SQLiteRepository) GetSheet(id int64) (model.TestSheet, error) {
	row := r.db.QueryRow(`SELECT id, plan_id, COALESCE(group_id, 0), name, description, prerequisites, config, command, notes, action, expected_result, execution_order, mockup_settings, created_at, updated_at
		FROM test_sheets WHERE id = ?`, id)
	sheet, err := scanSheet(row)
	if err != nil {
		return model.TestSheet{}, err
	}
	sheet.Steps, err = r.ListSteps(id)
	if err != nil {
		return model.TestSheet{}, err
	}
	sheet.Documents, err = r.ListSheetDocuments(id)
	return sheet, err
}

func (r *SQLiteRepository) UpdateSheet(id int64, input model.SheetInput) (model.TestSheet, error) {
	res, err := r.db.Exec(`UPDATE test_sheets SET name = ?, description = ?, prerequisites = ?, config = ?, command = ?, notes = ?, action = ?, expected_result = ?, execution_order = ?, mockup_settings = ?, updated_at = ? WHERE id = ?`,
		input.Name, input.Description, input.Prerequisites, input.Config, input.Command, input.Notes, input.Action, input.ExpectedResult, input.ExecutionOrder, input.MockupSettings, time.Now().UTC(), id)
	if err != nil {
		return model.TestSheet{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestSheet{}, sql.ErrNoRows
	}
	return r.GetSheet(id)
}

func (r *SQLiteRepository) DeleteSheet(id int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer rollback(tx)
	var groupID int64
	if err := tx.QueryRow(`SELECT COALESCE(group_id, 0) FROM test_sheets WHERE id = ?`, id).Scan(&groupID); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM test_sheets WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	if err := normalizeSheetOrderTx(tx, groupID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) CreateStep(sheetID int64, input model.StepInput) (model.TestSheetStep, error) {
	if input.ExecutionOrder == 0 {
		next, err := r.nextStepOrder(sheetID)
		if err != nil {
			return model.TestSheetStep{}, err
		}
		input.ExecutionOrder = next
	}
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO test_sheet_steps (sheet_id, action, field, expected_result, execution_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sheetID, input.Action, input.Field, input.ExpectedResult, input.ExecutionOrder, now, now)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.TestSheetStep{}, err
	}
	return r.GetStep(id)
}

func (r *SQLiteRepository) ListSteps(sheetID int64) ([]model.TestSheetStep, error) {
	rows, err := r.db.Query(`SELECT id, sheet_id, action, field, expected_result, execution_order, created_at, updated_at
		FROM test_sheet_steps WHERE sheet_id = ? ORDER BY execution_order ASC, id ASC`, sheetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	steps := []model.TestSheetStep{}
	for rows.Next() {
		step, err := scanStep(rows)
		if err != nil {
			return nil, err
		}
		documents, err := r.ListStepDocuments(step.ID)
		if err != nil {
			return nil, err
		}
		step.Documents = documents
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func (r *SQLiteRepository) GetStep(id int64) (model.TestSheetStep, error) {
	row := r.db.QueryRow(`SELECT id, sheet_id, action, field, expected_result, execution_order, created_at, updated_at
		FROM test_sheet_steps WHERE id = ?`, id)
	step, err := scanStep(row)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	step.Documents, err = r.ListStepDocuments(id)
	return step, err
}

func (r *SQLiteRepository) UpdateStep(id int64, input model.StepInput) (model.TestSheetStep, error) {
	res, err := r.db.Exec(`UPDATE test_sheet_steps SET action = ?, field = ?, expected_result = ?, execution_order = ?, updated_at = ? WHERE id = ?`,
		input.Action, input.Field, input.ExpectedResult, input.ExecutionOrder, time.Now().UTC(), id)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestSheetStep{}, sql.ErrNoRows
	}
	return r.GetStep(id)
}

func (r *SQLiteRepository) DeleteStep(id int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer rollback(tx)
	var sheetID int64
	if err := tx.QueryRow(`SELECT sheet_id FROM test_sheet_steps WHERE id = ?`, id).Scan(&sheetID); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM test_sheet_steps WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	if err := normalizeStepOrderTx(tx, sheetID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) DuplicateStep(id int64) (model.TestSheetStep, error) {
	step, err := r.GetStep(id)
	if err != nil {
		return model.TestSheetStep{}, err
	}
	return r.CreateStep(step.SheetID, model.StepInput{
		Action:         step.Action,
		Field:          step.Field,
		ExpectedResult: step.ExpectedResult,
	})
}

func (r *SQLiteRepository) ReorderSteps(sheetID int64, stepIDs []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer rollback(tx)
	for index, stepID := range stepIDs {
		res, err := tx.Exec(`UPDATE test_sheet_steps SET execution_order = ?, updated_at = ? WHERE id = ? AND sheet_id = ?`,
			index+1, time.Now().UTC(), stepID, sheetID)
		if err != nil {
			return err
		}
		if changed, _ := res.RowsAffected(); changed == 0 {
			return sql.ErrNoRows
		}
	}
	if err := normalizeStepOrderTx(tx, sheetID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) ReorderSheets(planID int64, sheetIDs []int64) error {
	groupID, err := r.DefaultGroupID(planID)
	if err != nil {
		return err
	}
	return r.ReorderGroupSheets(groupID, sheetIDs)
}

func (r *SQLiteRepository) ReorderGroupSheets(groupID int64, sheetIDs []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer rollback(tx)
	for index, sheetID := range sheetIDs {
		res, err := tx.Exec(`UPDATE test_sheets SET execution_order = ?, updated_at = ? WHERE id = ? AND group_id = ?`,
			index+1, time.Now().UTC(), sheetID, groupID)
		if err != nil {
			return err
		}
		if changed, _ := res.RowsAffected(); changed == 0 {
			return sql.ErrNoRows
		}
	}
	if err := normalizeSheetOrderTx(tx, groupID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) ReindexGroupSheets(groupID int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err := normalizeSheetOrderTx(tx, groupID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) ReindexSheetSteps(sheetID int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer rollback(tx)
	if err := normalizeStepOrderTx(tx, sheetID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLiteRepository) ListDocuments(planID int64) ([]model.TestDocument, error) {
	rows, err := r.db.Query(`SELECT id, plan_id, original_name, stored_name, storage_path, mime_type, size_bytes, sha256, description, created_at
		FROM test_documents WHERE plan_id = ? ORDER BY created_at DESC, id DESC`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func (r *SQLiteRepository) GetDocument(id int64) (model.TestDocument, error) {
	row := r.db.QueryRow(`SELECT id, plan_id, original_name, stored_name, storage_path, mime_type, size_bytes, sha256, description, created_at
		FROM test_documents WHERE id = ?`, id)
	return scanDocument(row)
}

func (r *SQLiteRepository) CreateDocument(input model.TestDocument) (model.TestDocument, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO test_documents
		(plan_id, original_name, stored_name, storage_path, mime_type, size_bytes, sha256, description, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		input.PlanID, input.OriginalName, input.StoredName, input.StoragePath, input.MimeType, input.SizeBytes, input.SHA256, input.Description, now)
	if err != nil {
		return model.TestDocument{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.TestDocument{}, err
	}
	return r.GetDocument(id)
}

func (r *SQLiteRepository) UpdateDocumentFile(id int64, storedName, storagePath, mimeType string, sizeBytes int64, sha256 string) (model.TestDocument, error) {
	res, err := r.db.Exec(`UPDATE test_documents SET stored_name = ?, storage_path = ?, mime_type = ?, size_bytes = ?, sha256 = ? WHERE id = ?`,
		storedName, storagePath, mimeType, sizeBytes, sha256, id)
	if err != nil {
		return model.TestDocument{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestDocument{}, sql.ErrNoRows
	}
	return r.GetDocument(id)
}

func (r *SQLiteRepository) DeleteDocument(id int64) (model.TestDocument, error) {
	document, err := r.GetDocument(id)
	if err != nil {
		return model.TestDocument{}, err
	}
	res, err := r.db.Exec(`DELETE FROM test_documents WHERE id = ?`, id)
	if err != nil {
		return model.TestDocument{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestDocument{}, sql.ErrNoRows
	}
	return document, nil
}

func (r *SQLiteRepository) LinkSheetDocument(sheetID, documentID int64) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(`INSERT OR IGNORE INTO test_sheet_documents (sheet_id, document_id, created_at) VALUES (?, ?, ?)`, sheetID, documentID, now)
	return err
}

func (r *SQLiteRepository) UnlinkSheetDocument(sheetID, documentID int64) error {
	_, err := r.db.Exec(`DELETE FROM test_sheet_documents WHERE sheet_id = ? AND document_id = ?`, sheetID, documentID)
	return err
}

func (r *SQLiteRepository) ListSheetDocuments(sheetID int64) ([]model.TestDocument, error) {
	rows, err := r.db.Query(`SELECT d.id, d.plan_id, d.original_name, d.stored_name, d.storage_path, d.mime_type, d.size_bytes, d.sha256, d.description, d.created_at
		FROM test_documents d
		INNER JOIN test_sheet_documents sd ON sd.document_id = d.id
		WHERE sd.sheet_id = ?
		ORDER BY d.original_name ASC, d.id ASC`, sheetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func (r *SQLiteRepository) LinkStepDocument(stepID, documentID int64) error {
	now := time.Now().UTC()
	_, err := r.db.Exec(`INSERT OR IGNORE INTO test_step_documents (step_id, document_id, created_at) VALUES (?, ?, ?)`, stepID, documentID, now)
	return err
}

func (r *SQLiteRepository) UnlinkStepDocument(stepID, documentID int64) error {
	_, err := r.db.Exec(`DELETE FROM test_step_documents WHERE step_id = ? AND document_id = ?`, stepID, documentID)
	return err
}

func (r *SQLiteRepository) ListStepDocuments(stepID int64) ([]model.TestDocument, error) {
	rows, err := r.db.Query(`SELECT d.id, d.plan_id, d.original_name, d.stored_name, d.storage_path, d.mime_type, d.size_bytes, d.sha256, d.description, d.created_at
		FROM test_documents d
		INNER JOIN test_step_documents sd ON sd.document_id = d.id
		WHERE sd.step_id = ?
		ORDER BY d.original_name ASC, d.id ASC`, stepID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDocuments(rows)
}

func (r *SQLiteRepository) CreateRunWithSnapshot(planID int64) (model.TestRun, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return model.TestRun{}, err
	}
	defer rollback(tx)

	var planName string
	if err := tx.QueryRow(`SELECT name FROM test_plans WHERE id = ?`, planID).Scan(&planName); err != nil {
		return model.TestRun{}, err
	}
	now := time.Now().UTC()
	res, err := tx.Exec(`INSERT INTO test_runs (plan_id, group_id, plan_name, group_name, status, started_at) VALUES (?, NULL, ?, '', ?, ?)`, planID, planName, model.TestRunStatusRunning, now)
	if err != nil {
		return model.TestRun{}, err
	}
	runID, err := res.LastInsertId()
	if err != nil {
		return model.TestRun{}, err
	}
	groupRows, err := tx.Query(`SELECT id, name, description, execution_order FROM test_plan_groups WHERE plan_id = ? ORDER BY execution_order ASC, id ASC`, planID)
	if err != nil {
		return model.TestRun{}, err
	}
	for groupRows.Next() {
		var group model.TestGroup
		if err := groupRows.Scan(&group.ID, &group.Name, &group.Description, &group.ExecutionOrder); err != nil {
			_ = groupRows.Close()
			return model.TestRun{}, err
		}
		runGroupRes, err := tx.Exec(`INSERT INTO test_run_groups (run_id, source_group_id, name, description, execution_order, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, runID, group.ID, group.Name, group.Description, group.ExecutionOrder, now)
		if err != nil {
			_ = groupRows.Close()
			return model.TestRun{}, err
		}
		runGroupID, err := runGroupRes.LastInsertId()
		if err != nil {
			_ = groupRows.Close()
			return model.TestRun{}, err
		}
		if _, err := tx.Exec(`INSERT INTO test_run_sheets
			(run_id, run_group_id, source_sheet_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, status, created_at, updated_at)
			SELECT ?, ?, id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, ?, ?, ?
			FROM test_sheets WHERE group_id = ? ORDER BY execution_order ASC, id ASC`,
			runID, runGroupID, model.RunSheetStatusPending, now, now, group.ID); err != nil {
			_ = groupRows.Close()
			return model.TestRun{}, err
		}
		runSheetRows, err := tx.Query(`SELECT id, source_sheet_id FROM test_run_sheets WHERE run_id = ? AND run_group_id = ?`, runID, runGroupID)
		if err != nil {
			_ = groupRows.Close()
			return model.TestRun{}, err
		}
		for runSheetRows.Next() {
			var runSheetID, sourceSheetID int64
			if err := runSheetRows.Scan(&runSheetID, &sourceSheetID); err != nil {
				_ = runSheetRows.Close()
				_ = groupRows.Close()
				return model.TestRun{}, err
			}
			if _, err := tx.Exec(`INSERT INTO test_run_steps
				(run_sheet_id, source_step_id, action, field, expected_result, execution_order, status, created_at, updated_at)
				SELECT ?, id, action, field, expected_result, execution_order, ?, ?, ?
				FROM test_sheet_steps WHERE sheet_id = ? ORDER BY execution_order ASC, id ASC`,
				runSheetID, model.RunSheetStatusPending, now, now, sourceSheetID); err != nil {
				_ = runSheetRows.Close()
				_ = groupRows.Close()
				return model.TestRun{}, err
			}
		}
		if err := runSheetRows.Err(); err != nil {
			_ = runSheetRows.Close()
			_ = groupRows.Close()
			return model.TestRun{}, err
		}
		_ = runSheetRows.Close()
	}
	if err := groupRows.Err(); err != nil {
		_ = groupRows.Close()
		return model.TestRun{}, err
	}
	_ = groupRows.Close()
	if err := tx.Commit(); err != nil {
		return model.TestRun{}, err
	}
	return r.GetRun(runID)
}

func (r *SQLiteRepository) CreateRunWithGroupSnapshot(groupID int64) (model.TestRun, error) {
	group, err := r.GetGroup(groupID)
	if err != nil {
		return model.TestRun{}, err
	}
	return r.CreateRunWithSnapshot(group.PlanID)
}

func (r *SQLiteRepository) GetRun(runID int64) (model.TestRun, error) {
	row := r.db.QueryRow(`SELECT r.id,
		(SELECT COUNT(*) FROM test_runs numbered
			WHERE numbered.plan_id = r.plan_id
				AND (numbered.started_at < r.started_at OR (numbered.started_at = r.started_at AND numbered.id <= r.id))) AS run_number,
		r.plan_id, COALESCE(r.group_id, 0), r.plan_name, r.group_name, r.status, r.started_at, r.finished_at
		FROM test_runs r WHERE r.id = ?`, runID)
	run, err := scanRun(row)
	if err != nil {
		return model.TestRun{}, err
	}
	sheets, err := r.ListRunSheets(runID)
	if err != nil {
		return model.TestRun{}, err
	}
	run.Sheets = sheets
	groups, err := r.ListRunGroups(runID)
	if err != nil {
		return model.TestRun{}, err
	}
	run.Groups = groups
	return run, nil
}

func (r *SQLiteRepository) ListPlanRuns(planID int64) ([]model.TestRunSummary, error) {
	return r.listRunSummaries(`WHERE r.plan_id = ?`, planID)
}

func (r *SQLiteRepository) ListGroupRuns(groupID int64) ([]model.TestRunSummary, error) {
	return r.listRunSummaries(`WHERE r.group_id = ?`, groupID)
}

func (r *SQLiteRepository) ListRunSummaries() ([]model.TestRunSummary, error) {
	return r.listRunSummaries(``, nil)
}

func (r *SQLiteRepository) ListPlanSummaries(includeDeleted bool) ([]model.TestPlanSummary, error) {
	plans, err := r.listPlans(includeDeleted)
	if err != nil {
		return nil, err
	}
	summaries := make([]model.TestPlanSummary, 0, len(plans))
	for _, plan := range plans {
		runs, err := r.ListPlanRuns(plan.ID)
		if err != nil {
			return nil, err
		}
		var sheetCount int
		if err := r.db.QueryRow(`SELECT COUNT(*) FROM test_sheets WHERE plan_id = ?`, plan.ID).Scan(&sheetCount); err != nil {
			return nil, err
		}
		var groupCount int
		if err := r.db.QueryRow(`SELECT COUNT(*) FROM test_plan_groups WHERE plan_id = ?`, plan.ID).Scan(&groupCount); err != nil {
			return nil, err
		}
		summary := model.TestPlanSummary{
			ID:          plan.ID,
			Name:        plan.Name,
			Description: plan.Description,
			Status:      model.TestRunStatusPending,
			SheetCount:  sheetCount,
			GroupCount:  groupCount,
			RunCount:    len(runs),
			UpdatedAt:   plan.UpdatedAt,
			DeletedAt:   plan.DeletedAt,
		}
		if len(runs) > 0 {
			latest := runs[0]
			summary.LatestRun = &latest
			summary.Status = normalizeRunStatus(latest.Status)
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (r *SQLiteRepository) listPlans(includeDeleted bool) ([]model.TestPlan, error) {
	query := `SELECT id, name, description, mockup_settings, created_at, updated_at, deleted_at FROM test_plans`
	if !includeDeleted {
		query += ` WHERE deleted_at IS NULL`
	}
	query += ` ORDER BY updated_at DESC, id DESC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	plans := []model.TestPlan{}
	for rows.Next() {
		plan, err := scanPlan(rows)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}
	return plans, rows.Err()
}

func (r *SQLiteRepository) listRunSummaries(where string, arg any) ([]model.TestRunSummary, error) {
	query := `WITH numbered_runs AS (
			SELECT id, ROW_NUMBER() OVER (PARTITION BY plan_id ORDER BY started_at ASC, id ASC) AS run_number
			FROM test_runs
		)
		SELECT r.id, nr.run_number, r.plan_id, COALESCE(r.group_id, 0), r.plan_name, r.group_name, r.status, r.started_at, r.finished_at,
		COUNT(DISTINCT rs.id) AS total_sheets,
		COUNT(rst.id) AS total_steps,
		COALESCE(SUM(CASE WHEN rst.status = 'pending' THEN 1 ELSE 0 END), 0) AS pending_steps,
		COALESCE(SUM(CASE WHEN rst.status = 'passed' THEN 1 ELSE 0 END), 0) AS passed_steps,
		COALESCE(SUM(CASE WHEN rst.status = 'failed' THEN 1 ELSE 0 END), 0) AS failed_steps,
		COALESCE(SUM(CASE WHEN rst.status = 'blocked' THEN 1 ELSE 0 END), 0) AS blocked_steps,
		COALESCE(SUM(CASE WHEN rst.status = 'skipped' THEN 1 ELSE 0 END), 0) AS skipped_steps
		FROM test_runs r
		JOIN numbered_runs nr ON nr.id = r.id
		LEFT JOIN test_run_sheets rs ON rs.run_id = r.id
		LEFT JOIN test_run_steps rst ON rst.run_sheet_id = rs.id ` + where + `
		GROUP BY r.id
		ORDER BY r.started_at DESC, r.id DESC`
	var rows *sql.Rows
	var err error
	if where == "" {
		rows, err = r.db.Query(query)
	} else {
		rows, err = r.db.Query(query, arg)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	summaries := []model.TestRunSummary{}
	for rows.Next() {
		summary, err := scanRunSummary(rows)
		if err != nil {
			return nil, err
		}
		if err := r.hydrateRunGroupProgress(&summary); err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

func (r *SQLiteRepository) hydrateRunGroupProgress(summary *model.TestRunSummary) error {
	groups, err := r.ListRunGroups(summary.ID)
	if err != nil {
		return err
	}
	summary.TotalGroups = len(groups)
	for _, group := range groups {
		switch runGroupStatus(group.Sheets) {
		case model.RunSheetStatusPassed:
			summary.PassedGroups++
		case model.RunSheetStatusFailed:
			summary.FailedGroups++
		case model.RunSheetStatusBlocked:
			summary.BlockedGroups++
		case model.RunSheetStatusSkipped:
			summary.SkippedGroups++
		default:
			summary.PendingGroups++
		}
	}
	return nil
}

func runGroupStatus(sheets []model.RunSheet) string {
	if len(sheets) == 0 {
		return model.RunSheetStatusPending
	}
	allSkipped := true
	for _, sheet := range sheets {
		switch runSheetStatus(sheet) {
		case model.RunSheetStatusFailed:
			return model.RunSheetStatusFailed
		case model.RunSheetStatusBlocked:
			return model.RunSheetStatusBlocked
		case model.RunSheetStatusPending:
			return model.RunSheetStatusPending
		case model.RunSheetStatusPassed:
			allSkipped = false
		}
	}
	if allSkipped {
		return model.RunSheetStatusSkipped
	}
	return model.RunSheetStatusPassed
}

func runSheetStatus(sheet model.RunSheet) string {
	if len(sheet.Steps) == 0 {
		return sheet.Status
	}
	nonSkipped := 0
	allPassed := true
	for _, step := range sheet.Steps {
		if step.Status == model.RunSheetStatusSkipped {
			continue
		}
		nonSkipped++
		switch step.Status {
		case model.RunSheetStatusFailed:
			return model.RunSheetStatusFailed
		case model.RunSheetStatusBlocked:
			return model.RunSheetStatusBlocked
		case model.RunSheetStatusPending:
			return model.RunSheetStatusPending
		case model.RunSheetStatusPassed:
		default:
			allPassed = false
		}
	}
	if nonSkipped == 0 {
		return model.RunSheetStatusSkipped
	}
	if allPassed {
		return model.RunSheetStatusPassed
	}
	return sheet.Status
}

func (r *SQLiteRepository) ReplayRun(runID int64) (model.TestRun, error) {
	source, err := r.GetRun(runID)
	if err != nil {
		return model.TestRun{}, err
	}
	return r.CreateRunWithSnapshot(source.PlanID)
}

func (r *SQLiteRepository) ArchiveRun(runID int64) (model.TestRun, error) {
	res, err := r.db.Exec(`UPDATE test_runs SET status = ? WHERE id = ?`, model.TestRunStatusArchived, runID)
	if err != nil {
		return model.TestRun{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestRun{}, sql.ErrNoRows
	}
	return r.GetRun(runID)
}

func (r *SQLiteRepository) CancelRun(runID int64) (model.TestRun, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`UPDATE test_runs SET status = ?, finished_at = ? WHERE id = ? AND status = ?`,
		model.TestRunStatusCanceled, now, runID, model.TestRunStatusRunning)
	if err != nil {
		return model.TestRun{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestRun{}, sql.ErrNoRows
	}
	return r.GetRun(runID)
}

func (r *SQLiteRepository) ListRunSheets(runID int64) ([]model.RunSheet, error) {
	rows, err := r.db.Query(`SELECT id, run_id, COALESCE(run_group_id, 0), source_sheet_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, status, actual_result, comment, created_at, updated_at
		FROM test_run_sheets WHERE run_id = ? ORDER BY execution_order ASC, id ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sheets := []model.RunSheet{}
	for rows.Next() {
		sheet, err := scanRunSheet(rows)
		if err != nil {
			return nil, err
		}
		sheets = append(sheets, sheet)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for index := range sheets {
		steps, err := r.ListRunSteps(sheets[index].ID)
		if err != nil {
			return nil, err
		}
		sheets[index].Steps = steps
		evidences, err := r.ListRunSheetEvidences(sheets[index].ID)
		if err != nil {
			return nil, err
		}
		sheets[index].Evidences = evidences
		if sheets[index].SourceSheetID != nil {
			documents, err := r.ListSheetDocuments(*sheets[index].SourceSheetID)
			if err != nil {
				return nil, err
			}
			sheets[index].Documents = documents
		}
	}
	return sheets, nil
}

func (r *SQLiteRepository) ListRunGroups(runID int64) ([]model.RunGroup, error) {
	rows, err := r.db.Query(`SELECT id, run_id, source_group_id, name, description, execution_order, created_at
		FROM test_run_groups WHERE run_id = ? ORDER BY execution_order ASC, id ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	groups := []model.RunGroup{}
	for rows.Next() {
		group, err := scanRunGroup(rows)
		if err != nil {
			return nil, err
		}
		sheets, err := r.ListRunSheetsByGroup(group.ID)
		if err != nil {
			return nil, err
		}
		group.Sheets = sheets
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (r *SQLiteRepository) ListRunSheetsByGroup(runGroupID int64) ([]model.RunSheet, error) {
	rows, err := r.db.Query(`SELECT id, run_id, COALESCE(run_group_id, 0), source_sheet_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, status, actual_result, comment, created_at, updated_at
		FROM test_run_sheets WHERE run_group_id = ? ORDER BY execution_order ASC, id ASC`, runGroupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sheets := []model.RunSheet{}
	for rows.Next() {
		sheet, err := scanRunSheet(rows)
		if err != nil {
			return nil, err
		}
		sheets = append(sheets, sheet)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for index := range sheets {
		steps, err := r.ListRunSteps(sheets[index].ID)
		if err != nil {
			return nil, err
		}
		sheets[index].Steps = steps
		evidences, err := r.ListRunSheetEvidences(sheets[index].ID)
		if err != nil {
			return nil, err
		}
		sheets[index].Evidences = evidences
		if sheets[index].SourceSheetID != nil {
			documents, err := r.ListSheetDocuments(*sheets[index].SourceSheetID)
			if err != nil {
				return nil, err
			}
			sheets[index].Documents = documents
		}
	}
	return sheets, nil
}

func (r *SQLiteRepository) UpdateRunSheet(runID, runSheetID int64, input model.RunSheetResultInput) (model.RunSheet, error) {
	res, err := r.db.Exec(`UPDATE test_run_sheets SET status = ?, actual_result = ?, comment = ?, updated_at = ? WHERE id = ? AND run_id = ?`,
		input.Status, input.ActualResult, input.Comment, time.Now().UTC(), runSheetID, runID)
	if err != nil {
		return model.RunSheet{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.RunSheet{}, sql.ErrNoRows
	}
	row := r.db.QueryRow(`SELECT id, run_id, COALESCE(run_group_id, 0), source_sheet_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, status, actual_result, comment, created_at, updated_at
		FROM test_run_sheets WHERE id = ? AND run_id = ?`, runSheetID, runID)
	return scanRunSheet(row)
}

func (r *SQLiteRepository) ListRunSteps(runSheetID int64) ([]model.RunStep, error) {
	rows, err := r.db.Query(`SELECT id, run_sheet_id, source_step_id, action, field, expected_result, execution_order, status, actual_result, comment, created_at, updated_at
		FROM test_run_steps WHERE run_sheet_id = ? ORDER BY execution_order ASC, id ASC`, runSheetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	steps := []model.RunStep{}
	for rows.Next() {
		step, err := scanRunStep(rows)
		if err != nil {
			return nil, err
		}
		if step.SourceStepID != nil {
			documents, err := r.ListStepDocuments(*step.SourceStepID)
			if err != nil {
				return nil, err
			}
			step.Documents = documents
		}
		evidences, err := r.ListRunStepEvidences(step.ID)
		if err != nil {
			return nil, err
		}
		step.Evidences = evidences
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func (r *SQLiteRepository) UpdateRunStep(runID, runStepID int64, input model.RunStepResultInput) (model.RunStep, error) {
	res, err := r.db.Exec(`UPDATE test_run_steps SET status = ?, actual_result = ?, comment = ?, updated_at = ?
		WHERE id = ? AND run_sheet_id IN (SELECT id FROM test_run_sheets WHERE run_id = ?)`,
		input.Status, input.ActualResult, input.Comment, time.Now().UTC(), runStepID, runID)
	if err != nil {
		return model.RunStep{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.RunStep{}, sql.ErrNoRows
	}
	row := r.db.QueryRow(`SELECT id, run_sheet_id, source_step_id, action, field, expected_result, execution_order, status, actual_result, comment, created_at, updated_at
		FROM test_run_steps WHERE id = ?`, runStepID)
	return scanRunStep(row)
}

func (r *SQLiteRepository) GetRunIDForRunSheet(runSheetID int64) (int64, error) {
	var runID int64
	err := r.db.QueryRow(`SELECT run_id FROM test_run_sheets WHERE id = ?`, runSheetID).Scan(&runID)
	return runID, err
}

func (r *SQLiteRepository) GetRunIDForRunStep(runStepID int64) (int64, error) {
	var runID int64
	err := r.db.QueryRow(`SELECT rs.run_id
		FROM test_run_steps rst
		JOIN test_run_sheets rs ON rs.id = rst.run_sheet_id
		WHERE rst.id = ?`, runStepID).Scan(&runID)
	return runID, err
}

func (r *SQLiteRepository) ListRunSheetEvidences(runSheetID int64) ([]model.Evidence, error) {
	rows, err := r.db.Query(`SELECT id, run_sheet_id, name, path, mime_type, size_bytes, comment, created_at
		FROM test_run_evidences WHERE run_sheet_id = ? ORDER BY created_at DESC, id DESC`, runSheetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	evidences := []model.Evidence{}
	for rows.Next() {
		evidence, err := scanEvidence(rows)
		if err != nil {
			return nil, err
		}
		evidences = append(evidences, evidence)
	}
	return evidences, rows.Err()
}

func (r *SQLiteRepository) GetEvidence(evidenceID int64) (model.Evidence, error) {
	row := r.db.QueryRow(`SELECT id, run_sheet_id, name, path, mime_type, size_bytes, comment, created_at
		FROM test_run_evidences WHERE id = ?`, evidenceID)
	return scanEvidence(row)
}

func (r *SQLiteRepository) CreateEvidence(input model.Evidence) (model.Evidence, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO test_run_evidences
		(run_sheet_id, name, path, mime_type, size_bytes, comment, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		input.RunSheetID, input.Name, input.Path, input.MimeType, input.SizeBytes, input.Comment, now)
	if err != nil {
		return model.Evidence{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Evidence{}, err
	}
	return r.GetEvidence(id)
}

func (r *SQLiteRepository) UpdateEvidenceFile(evidenceID int64, storagePath, mimeType string, sizeBytes int64) (model.Evidence, error) {
	res, err := r.db.Exec(`UPDATE test_run_evidences SET path = ?, mime_type = ?, size_bytes = ? WHERE id = ?`,
		storagePath, mimeType, sizeBytes, evidenceID)
	if err != nil {
		return model.Evidence{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.Evidence{}, sql.ErrNoRows
	}
	return r.GetEvidence(evidenceID)
}

func (r *SQLiteRepository) DeleteEvidence(evidenceID int64) (model.Evidence, error) {
	evidence, err := r.GetEvidence(evidenceID)
	if err != nil {
		return model.Evidence{}, err
	}
	res, err := r.db.Exec(`DELETE FROM test_run_evidences WHERE id = ?`, evidenceID)
	if err != nil {
		return model.Evidence{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.Evidence{}, sql.ErrNoRows
	}
	return evidence, nil
}

func (r *SQLiteRepository) ListRunStepEvidences(runStepID int64) ([]model.Evidence, error) {
	rows, err := r.db.Query(`SELECT id, run_step_id, name, path, mime_type, size_bytes, created_at
		FROM test_run_step_evidences WHERE run_step_id = ? ORDER BY created_at DESC, id DESC`, runStepID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	evidences := []model.Evidence{}
	for rows.Next() {
		evidence, err := scanStepEvidence(rows)
		if err != nil {
			return nil, err
		}
		evidences = append(evidences, evidence)
	}
	return evidences, rows.Err()
}

func (r *SQLiteRepository) GetStepEvidence(evidenceID int64) (model.Evidence, error) {
	row := r.db.QueryRow(`SELECT id, run_step_id, name, path, mime_type, size_bytes, created_at
		FROM test_run_step_evidences WHERE id = ?`, evidenceID)
	return scanStepEvidence(row)
}

func (r *SQLiteRepository) CreateStepEvidence(input model.Evidence) (model.Evidence, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO test_run_step_evidences
		(run_step_id, name, path, mime_type, size_bytes, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		input.RunStepID, input.Name, input.Path, input.MimeType, input.SizeBytes, now)
	if err != nil {
		return model.Evidence{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Evidence{}, err
	}
	return r.GetStepEvidence(id)
}

func (r *SQLiteRepository) UpdateStepEvidenceFile(evidenceID int64, storagePath, mimeType string, sizeBytes int64) (model.Evidence, error) {
	res, err := r.db.Exec(`UPDATE test_run_step_evidences SET path = ?, mime_type = ?, size_bytes = ? WHERE id = ?`,
		storagePath, mimeType, sizeBytes, evidenceID)
	if err != nil {
		return model.Evidence{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.Evidence{}, sql.ErrNoRows
	}
	return r.GetStepEvidence(evidenceID)
}

func (r *SQLiteRepository) DeleteStepEvidence(evidenceID int64) (model.Evidence, error) {
	evidence, err := r.GetStepEvidence(evidenceID)
	if err != nil {
		return model.Evidence{}, err
	}
	res, err := r.db.Exec(`DELETE FROM test_run_step_evidences WHERE id = ?`, evidenceID)
	if err != nil {
		return model.Evidence{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.Evidence{}, sql.ErrNoRows
	}
	return evidence, nil
}

func (r *SQLiteRepository) FinishRun(runID int64) (model.TestRun, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`UPDATE test_runs SET status = ?, finished_at = ? WHERE id = ? AND status = ?`,
		model.TestRunStatusCompleted, now, runID, model.TestRunStatusRunning)
	if err != nil {
		return model.TestRun{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestRun{}, sql.ErrNoRows
	}
	return r.GetRun(runID)
}

func (r *SQLiteRepository) nextGroupOrder(planID int64) (int, error) {
	var next sql.NullInt64
	if err := r.db.QueryRow(`SELECT MAX(execution_order) + 1 FROM test_plan_groups WHERE plan_id = ?`, planID).Scan(&next); err != nil {
		return 0, err
	}
	if !next.Valid {
		return 1, nil
	}
	return int(next.Int64), nil
}

func (r *SQLiteRepository) nextSheetOrder(groupID int64) (int, error) {
	var next sql.NullInt64
	if err := r.db.QueryRow(`SELECT MAX(execution_order) + 1 FROM test_sheets WHERE group_id = ?`, groupID).Scan(&next); err != nil {
		return 0, err
	}
	if !next.Valid {
		return 1, nil
	}
	return int(next.Int64), nil
}

func (r *SQLiteRepository) nextStepOrder(sheetID int64) (int, error) {
	var next sql.NullInt64
	if err := r.db.QueryRow(`SELECT MAX(execution_order) + 1 FROM test_sheet_steps WHERE sheet_id = ?`, sheetID).Scan(&next); err != nil {
		return 0, err
	}
	if !next.Valid {
		return 1, nil
	}
	return int(next.Int64), nil
}

func normalizeSheetOrderTx(tx *sql.Tx, groupID int64) error {
	rows, err := tx.Query(`SELECT id FROM test_sheets WHERE group_id = ? ORDER BY execution_order ASC, id ASC`, groupID)
	if err != nil {
		return err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	for index, id := range ids {
		if _, err := tx.Exec(`UPDATE test_sheets SET execution_order = ?, updated_at = ? WHERE id = ?`, index+1, now, id); err != nil {
			return err
		}
	}
	return nil
}

func normalizeStepOrderTx(tx *sql.Tx, sheetID int64) error {
	rows, err := tx.Query(`SELECT id FROM test_sheet_steps WHERE sheet_id = ? ORDER BY execution_order ASC, id ASC`, sheetID)
	if err != nil {
		return err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	for index, id := range ids {
		if _, err := tx.Exec(`UPDATE test_sheet_steps SET execution_order = ?, updated_at = ? WHERE id = ?`, index+1, now, id); err != nil {
			return err
		}
	}
	return nil
}

func scanPlan(scanner interface{ Scan(...any) error }) (model.TestPlan, error) {
	var plan model.TestPlan
	var deleted sql.NullTime
	err := scanner.Scan(&plan.ID, &plan.Name, &plan.Description, &plan.MockupSettings, &plan.CreatedAt, &plan.UpdatedAt, &deleted)
	if deleted.Valid {
		plan.DeletedAt = &deleted.Time
	}
	return plan, err
}

func scanGroup(scanner interface{ Scan(...any) error }) (model.TestGroup, error) {
	var group model.TestGroup
	err := scanner.Scan(&group.ID, &group.PlanID, &group.Name, &group.Description, &group.ExecutionOrder, &group.CreatedAt, &group.UpdatedAt)
	return group, err
}

func scanGroupWithSheetCount(scanner interface{ Scan(...any) error }) (model.TestGroup, error) {
	var group model.TestGroup
	err := scanner.Scan(&group.ID, &group.PlanID, &group.Name, &group.Description, &group.ExecutionOrder, &group.CreatedAt, &group.UpdatedAt, &group.SheetCount)
	return group, err
}

func scanSheets(rows *sql.Rows) ([]model.TestSheet, error) {
	sheets := []model.TestSheet{}
	for rows.Next() {
		sheet, err := scanSheet(rows)
		if err != nil {
			return nil, err
		}
		sheets = append(sheets, sheet)
	}
	return sheets, rows.Err()
}

func scanSheet(scanner interface{ Scan(...any) error }) (model.TestSheet, error) {
	var sheet model.TestSheet
	err := scanner.Scan(&sheet.ID, &sheet.PlanID, &sheet.GroupID, &sheet.Name, &sheet.Description, &sheet.Prerequisites, &sheet.Config, &sheet.Command, &sheet.Notes, &sheet.Action, &sheet.ExpectedResult, &sheet.ExecutionOrder, &sheet.MockupSettings, &sheet.CreatedAt, &sheet.UpdatedAt)
	return sheet, err
}

func scanStep(scanner interface{ Scan(...any) error }) (model.TestSheetStep, error) {
	var step model.TestSheetStep
	err := scanner.Scan(&step.ID, &step.SheetID, &step.Action, &step.Field, &step.ExpectedResult, &step.ExecutionOrder, &step.CreatedAt, &step.UpdatedAt)
	return step, err
}

func scanDocuments(rows *sql.Rows) ([]model.TestDocument, error) {
	documents := []model.TestDocument{}
	for rows.Next() {
		document, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, rows.Err()
}

func scanDocument(scanner interface{ Scan(...any) error }) (model.TestDocument, error) {
	var document model.TestDocument
	err := scanner.Scan(&document.ID, &document.PlanID, &document.OriginalName, &document.StoredName, &document.StoragePath, &document.MimeType, &document.SizeBytes, &document.SHA256, &document.Description, &document.CreatedAt)
	return document, err
}

func scanRun(scanner interface{ Scan(...any) error }) (model.TestRun, error) {
	var run model.TestRun
	var finished sql.NullTime
	err := scanner.Scan(&run.ID, &run.RunNumber, &run.PlanID, &run.GroupID, &run.PlanName, &run.GroupName, &run.Status, &run.StartedAt, &finished)
	if finished.Valid {
		run.FinishedAt = &finished.Time
	}
	run.Status = normalizeRunStatus(run.Status)
	return run, err
}

func scanRunSummary(scanner interface{ Scan(...any) error }) (model.TestRunSummary, error) {
	var summary model.TestRunSummary
	var finished sql.NullTime
	err := scanner.Scan(
		&summary.ID,
		&summary.RunNumber,
		&summary.PlanID,
		&summary.GroupID,
		&summary.PlanName,
		&summary.GroupName,
		&summary.Status,
		&summary.StartedAt,
		&finished,
		&summary.TotalSheets,
		&summary.TotalSteps,
		&summary.PendingSteps,
		&summary.PassedSteps,
		&summary.FailedSteps,
		&summary.BlockedSteps,
		&summary.SkippedSteps,
	)
	if finished.Valid {
		summary.FinishedAt = &finished.Time
	}
	summary.Status = normalizeRunStatus(summary.Status)
	return summary, err
}

func scanRunGroup(scanner interface{ Scan(...any) error }) (model.RunGroup, error) {
	var group model.RunGroup
	var sourceGroupID sql.NullInt64
	err := scanner.Scan(&group.ID, &group.RunID, &sourceGroupID, &group.Name, &group.Description, &group.ExecutionOrder, &group.CreatedAt)
	if sourceGroupID.Valid {
		value := sourceGroupID.Int64
		group.SourceGroupID = &value
	}
	return group, err
}

func normalizeRunStatus(status string) string {
	if status == "finished" {
		return model.TestRunStatusCompleted
	}
	return status
}

func scanRunSheet(scanner interface{ Scan(...any) error }) (model.RunSheet, error) {
	var sheet model.RunSheet
	var sourceSheetID sql.NullInt64
	err := scanner.Scan(&sheet.ID, &sheet.RunID, &sheet.RunGroupID, &sourceSheetID, &sheet.Name, &sheet.Description, &sheet.Prerequisites, &sheet.Config, &sheet.Command, &sheet.Notes, &sheet.Action, &sheet.ExpectedResult, &sheet.ExecutionOrder, &sheet.Status, &sheet.ActualResult, &sheet.Comment, &sheet.CreatedAt, &sheet.UpdatedAt)
	if sourceSheetID.Valid {
		value := sourceSheetID.Int64
		sheet.SourceSheetID = &value
	}
	return sheet, err
}

func scanEvidence(scanner interface{ Scan(...any) error }) (model.Evidence, error) {
	var evidence model.Evidence
	err := scanner.Scan(&evidence.ID, &evidence.RunSheetID, &evidence.Name, &evidence.Path, &evidence.MimeType, &evidence.SizeBytes, &evidence.Comment, &evidence.CreatedAt)
	return evidence, err
}

func scanStepEvidence(scanner interface{ Scan(...any) error }) (model.Evidence, error) {
	var evidence model.Evidence
	err := scanner.Scan(&evidence.ID, &evidence.RunStepID, &evidence.Name, &evidence.Path, &evidence.MimeType, &evidence.SizeBytes, &evidence.CreatedAt)
	return evidence, err
}

func scanRunStep(scanner interface{ Scan(...any) error }) (model.RunStep, error) {
	var step model.RunStep
	var sourceStepID sql.NullInt64
	err := scanner.Scan(&step.ID, &step.RunSheetID, &sourceStepID, &step.Action, &step.Field, &step.ExpectedResult, &step.ExecutionOrder, &step.Status, &step.ActualResult, &step.Comment, &step.CreatedAt, &step.UpdatedAt)
	if sourceStepID.Valid {
		value := sourceStepID.Int64
		step.SourceStepID = &value
	}
	return step, err
}

func rollback(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

const migrationSQL = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS test_plans (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	mockup_settings TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	deleted_at DATETIME
);

CREATE TABLE IF NOT EXISTS test_sheets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	plan_id INTEGER NOT NULL,
	group_id INTEGER,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	prerequisites TEXT NOT NULL DEFAULT '',
	config TEXT NOT NULL DEFAULT '',
	command TEXT NOT NULL DEFAULT '',
	notes TEXT NOT NULL DEFAULT '',
	action TEXT NOT NULL DEFAULT '',
	expected_result TEXT NOT NULL DEFAULT '',
	execution_order INTEGER NOT NULL DEFAULT 0,
	mockup_settings TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (plan_id) REFERENCES test_plans(id) ON DELETE CASCADE,
	FOREIGN KEY (group_id) REFERENCES test_plan_groups(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS test_plan_groups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	plan_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	execution_order INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (plan_id) REFERENCES test_plans(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS test_sheet_steps (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	sheet_id INTEGER NOT NULL,
	action TEXT NOT NULL DEFAULT '',
	field TEXT NOT NULL DEFAULT '',
	expected_result TEXT NOT NULL DEFAULT '',
	execution_order INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (sheet_id) REFERENCES test_sheets(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS test_attachments (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	sheet_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	path TEXT NOT NULL,
	mime_type TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	FOREIGN KEY (sheet_id) REFERENCES test_sheets(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS test_runs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	plan_id INTEGER NOT NULL,
	group_id INTEGER,
	plan_name TEXT NOT NULL,
	group_name TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'running',
	started_at DATETIME NOT NULL,
	finished_at DATETIME,
	FOREIGN KEY (plan_id) REFERENCES test_plans(id) ON DELETE CASCADE,
	FOREIGN KEY (group_id) REFERENCES test_plan_groups(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS test_run_sheets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_id INTEGER NOT NULL,
	run_group_id INTEGER,
	source_sheet_id INTEGER,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	prerequisites TEXT NOT NULL DEFAULT '',
	config TEXT NOT NULL DEFAULT '',
	command TEXT NOT NULL DEFAULT '',
	notes TEXT NOT NULL DEFAULT '',
	action TEXT NOT NULL DEFAULT '',
	expected_result TEXT NOT NULL DEFAULT '',
	execution_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'pending',
	actual_result TEXT NOT NULL DEFAULT '',
	comment TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (run_id) REFERENCES test_runs(id) ON DELETE CASCADE,
	FOREIGN KEY (run_group_id) REFERENCES test_run_groups(id) ON DELETE CASCADE,
	FOREIGN KEY (source_sheet_id) REFERENCES test_sheets(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS test_run_groups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_id INTEGER NOT NULL,
	source_group_id INTEGER,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	execution_order INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (run_id) REFERENCES test_runs(id) ON DELETE CASCADE,
	FOREIGN KEY (source_group_id) REFERENCES test_plan_groups(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS test_run_steps (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_sheet_id INTEGER NOT NULL,
	source_step_id INTEGER,
	action TEXT NOT NULL DEFAULT '',
	field TEXT NOT NULL DEFAULT '',
	expected_result TEXT NOT NULL DEFAULT '',
	execution_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'pending',
	actual_result TEXT NOT NULL DEFAULT '',
	comment TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (run_sheet_id) REFERENCES test_run_sheets(id) ON DELETE CASCADE,
	FOREIGN KEY (source_step_id) REFERENCES test_sheet_steps(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS test_run_evidences (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_sheet_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	path TEXT NOT NULL,
	mime_type TEXT NOT NULL DEFAULT '',
	size_bytes INTEGER NOT NULL DEFAULT 0,
	comment TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	FOREIGN KEY (run_sheet_id) REFERENCES test_run_sheets(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS test_run_step_evidences (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_step_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	path TEXT NOT NULL,
	mime_type TEXT NOT NULL DEFAULT '',
	size_bytes INTEGER NOT NULL DEFAULT 0,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (run_step_id) REFERENCES test_run_steps(id) ON DELETE CASCADE
);
`

const documentMigrationSQL = `
CREATE TABLE IF NOT EXISTS test_documents (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	plan_id INTEGER NOT NULL,
	original_name TEXT NOT NULL,
	stored_name TEXT NOT NULL,
	storage_path TEXT NOT NULL,
	mime_type TEXT NOT NULL DEFAULT '',
	size_bytes INTEGER NOT NULL DEFAULT 0,
	sha256 TEXT NOT NULL DEFAULT '',
	description TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	FOREIGN KEY (plan_id) REFERENCES test_plans(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS test_sheet_documents (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	sheet_id INTEGER NOT NULL,
	document_id INTEGER NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (sheet_id) REFERENCES test_sheets(id) ON DELETE CASCADE,
	FOREIGN KEY (document_id) REFERENCES test_documents(id) ON DELETE CASCADE,
	UNIQUE(sheet_id, document_id)
);

CREATE TABLE IF NOT EXISTS test_step_documents (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	step_id INTEGER NOT NULL,
	document_id INTEGER NOT NULL,
	created_at DATETIME NOT NULL,
	FOREIGN KEY (step_id) REFERENCES test_sheet_steps(id) ON DELETE CASCADE,
	FOREIGN KEY (document_id) REFERENCES test_documents(id) ON DELETE CASCADE,
	UNIQUE(step_id, document_id)
);
`
