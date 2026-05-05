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
	_, err := r.db.Exec(migrationSQL)
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
	return r.GetPlan(id)
}

func (r *SQLiteRepository) ListPlans() ([]model.TestPlan, error) {
	rows, err := r.db.Query(`SELECT id, name, description, mockup_settings, created_at, updated_at FROM test_plans ORDER BY updated_at DESC, id DESC`)
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
	row := r.db.QueryRow(`SELECT id, name, description, mockup_settings, created_at, updated_at FROM test_plans WHERE id = ?`, id)
	return scanPlan(row)
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
	res, err := r.db.Exec(`DELETE FROM test_plans WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLiteRepository) CreateSheet(planID int64, input model.SheetInput) (model.TestSheet, error) {
	if input.ExecutionOrder == 0 {
		next, err := r.nextSheetOrder(planID)
		if err != nil {
			return model.TestSheet{}, err
		}
		input.ExecutionOrder = next
	}
	now := time.Now().UTC()
	res, err := r.db.Exec(`INSERT INTO test_sheets
		(plan_id, name, description, prerequisites, action, expected_result, execution_order, mockup_settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		planID, input.Name, input.Description, input.Prerequisites, input.Action, input.ExpectedResult, input.ExecutionOrder, input.MockupSettings, now, now)
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
	rows, err := r.db.Query(`SELECT id, plan_id, name, description, prerequisites, action, expected_result, execution_order, mockup_settings, created_at, updated_at
		FROM test_sheets WHERE plan_id = ? ORDER BY execution_order ASC, id ASC`, planID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSheets(rows)
}

func (r *SQLiteRepository) GetSheet(id int64) (model.TestSheet, error) {
	row := r.db.QueryRow(`SELECT id, plan_id, name, description, prerequisites, action, expected_result, execution_order, mockup_settings, created_at, updated_at
		FROM test_sheets WHERE id = ?`, id)
	return scanSheet(row)
}

func (r *SQLiteRepository) UpdateSheet(id int64, input model.SheetInput) (model.TestSheet, error) {
	res, err := r.db.Exec(`UPDATE test_sheets SET name = ?, description = ?, prerequisites = ?, action = ?, expected_result = ?, execution_order = ?, mockup_settings = ?, updated_at = ? WHERE id = ?`,
		input.Name, input.Description, input.Prerequisites, input.Action, input.ExpectedResult, input.ExecutionOrder, input.MockupSettings, time.Now().UTC(), id)
	if err != nil {
		return model.TestSheet{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestSheet{}, sql.ErrNoRows
	}
	return r.GetSheet(id)
}

func (r *SQLiteRepository) DeleteSheet(id int64) error {
	res, err := r.db.Exec(`DELETE FROM test_sheets WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLiteRepository) ReorderSheets(planID int64, sheetIDs []int64) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer rollback(tx)
	for index, sheetID := range sheetIDs {
		res, err := tx.Exec(`UPDATE test_sheets SET execution_order = ?, updated_at = ? WHERE id = ? AND plan_id = ?`,
			index+1, time.Now().UTC(), sheetID, planID)
		if err != nil {
			return err
		}
		if changed, _ := res.RowsAffected(); changed == 0 {
			return sql.ErrNoRows
		}
	}
	return tx.Commit()
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
	res, err := tx.Exec(`INSERT INTO test_runs (plan_id, plan_name, status, started_at) VALUES (?, ?, ?, ?)`, planID, planName, "running", now)
	if err != nil {
		return model.TestRun{}, err
	}
	runID, err := res.LastInsertId()
	if err != nil {
		return model.TestRun{}, err
	}
	rows, err := tx.Query(`SELECT id, name, description, prerequisites, action, expected_result, execution_order
		FROM test_sheets WHERE plan_id = ? ORDER BY execution_order ASC, id ASC`, planID)
	if err != nil {
		return model.TestRun{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var sheet model.TestSheet
		if err := rows.Scan(&sheet.ID, &sheet.Name, &sheet.Description, &sheet.Prerequisites, &sheet.Action, &sheet.ExpectedResult, &sheet.ExecutionOrder); err != nil {
			return model.TestRun{}, err
		}
		_, err := tx.Exec(`INSERT INTO test_run_sheets
			(run_id, source_sheet_id, name, description, prerequisites, action, expected_result, execution_order, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			runID, sheet.ID, sheet.Name, sheet.Description, sheet.Prerequisites, sheet.Action, sheet.ExpectedResult, sheet.ExecutionOrder, model.RunSheetStatusPending, now, now)
		if err != nil {
			return model.TestRun{}, err
		}
	}
	if err := rows.Err(); err != nil {
		return model.TestRun{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.TestRun{}, err
	}
	return r.GetRun(runID)
}

func (r *SQLiteRepository) GetRun(runID int64) (model.TestRun, error) {
	row := r.db.QueryRow(`SELECT id, plan_id, plan_name, status, started_at, finished_at FROM test_runs WHERE id = ?`, runID)
	run, err := scanRun(row)
	if err != nil {
		return model.TestRun{}, err
	}
	sheets, err := r.ListRunSheets(runID)
	if err != nil {
		return model.TestRun{}, err
	}
	run.Sheets = sheets
	return run, nil
}

func (r *SQLiteRepository) ListRunSheets(runID int64) ([]model.RunSheet, error) {
	rows, err := r.db.Query(`SELECT id, run_id, source_sheet_id, name, description, prerequisites, action, expected_result, execution_order, status, actual_result, comment, created_at, updated_at
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
	return sheets, rows.Err()
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
	row := r.db.QueryRow(`SELECT id, run_id, source_sheet_id, name, description, prerequisites, action, expected_result, execution_order, status, actual_result, comment, created_at, updated_at
		FROM test_run_sheets WHERE id = ? AND run_id = ?`, runSheetID, runID)
	return scanRunSheet(row)
}

func (r *SQLiteRepository) FinishRun(runID int64) (model.TestRun, error) {
	now := time.Now().UTC()
	res, err := r.db.Exec(`UPDATE test_runs SET status = ?, finished_at = ? WHERE id = ?`, "finished", now, runID)
	if err != nil {
		return model.TestRun{}, err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return model.TestRun{}, sql.ErrNoRows
	}
	return r.GetRun(runID)
}

func (r *SQLiteRepository) nextSheetOrder(planID int64) (int, error) {
	var next sql.NullInt64
	if err := r.db.QueryRow(`SELECT MAX(execution_order) + 1 FROM test_sheets WHERE plan_id = ?`, planID).Scan(&next); err != nil {
		return 0, err
	}
	if !next.Valid {
		return 1, nil
	}
	return int(next.Int64), nil
}

func scanPlan(scanner interface{ Scan(...any) error }) (model.TestPlan, error) {
	var plan model.TestPlan
	err := scanner.Scan(&plan.ID, &plan.Name, &plan.Description, &plan.MockupSettings, &plan.CreatedAt, &plan.UpdatedAt)
	return plan, err
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
	err := scanner.Scan(&sheet.ID, &sheet.PlanID, &sheet.Name, &sheet.Description, &sheet.Prerequisites, &sheet.Action, &sheet.ExpectedResult, &sheet.ExecutionOrder, &sheet.MockupSettings, &sheet.CreatedAt, &sheet.UpdatedAt)
	return sheet, err
}

func scanRun(scanner interface{ Scan(...any) error }) (model.TestRun, error) {
	var run model.TestRun
	var finished sql.NullTime
	err := scanner.Scan(&run.ID, &run.PlanID, &run.PlanName, &run.Status, &run.StartedAt, &finished)
	if finished.Valid {
		run.FinishedAt = &finished.Time
	}
	return run, err
}

func scanRunSheet(scanner interface{ Scan(...any) error }) (model.RunSheet, error) {
	var sheet model.RunSheet
	var sourceSheetID sql.NullInt64
	err := scanner.Scan(&sheet.ID, &sheet.RunID, &sourceSheetID, &sheet.Name, &sheet.Description, &sheet.Prerequisites, &sheet.Action, &sheet.ExpectedResult, &sheet.ExecutionOrder, &sheet.Status, &sheet.ActualResult, &sheet.Comment, &sheet.CreatedAt, &sheet.UpdatedAt)
	if sourceSheetID.Valid {
		value := sourceSheetID.Int64
		sheet.SourceSheetID = &value
	}
	return sheet, err
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
	updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS test_sheets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	plan_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	prerequisites TEXT NOT NULL DEFAULT '',
	action TEXT NOT NULL DEFAULT '',
	expected_result TEXT NOT NULL DEFAULT '',
	execution_order INTEGER NOT NULL DEFAULT 0,
	mockup_settings TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (plan_id) REFERENCES test_plans(id) ON DELETE CASCADE
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
	plan_name TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'running',
	started_at DATETIME NOT NULL,
	finished_at DATETIME,
	FOREIGN KEY (plan_id) REFERENCES test_plans(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS test_run_sheets (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_id INTEGER NOT NULL,
	source_sheet_id INTEGER,
	name TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	prerequisites TEXT NOT NULL DEFAULT '',
	action TEXT NOT NULL DEFAULT '',
	expected_result TEXT NOT NULL DEFAULT '',
	execution_order INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'pending',
	actual_result TEXT NOT NULL DEFAULT '',
	comment TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL,
	FOREIGN KEY (run_id) REFERENCES test_runs(id) ON DELETE CASCADE,
	FOREIGN KEY (source_sheet_id) REFERENCES test_sheets(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS test_run_evidences (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_sheet_id INTEGER NOT NULL,
	name TEXT NOT NULL,
	path TEXT NOT NULL,
	mime_type TEXT NOT NULL DEFAULT '',
	comment TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	FOREIGN KEY (run_sheet_id) REFERENCES test_run_sheets(id) ON DELETE CASCADE
);
`
