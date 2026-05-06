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
	if err := r.ensureNullableDateTimeColumn("test_plans", "deleted_at"); err != nil {
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
		(plan_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, mockup_settings, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		planID, input.Name, input.Description, input.Prerequisites, input.Config, input.Command, input.Notes, input.Action, input.ExpectedResult, input.ExecutionOrder, input.MockupSettings, now, now)
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
	rows, err := r.db.Query(`SELECT id, plan_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, mockup_settings, created_at, updated_at
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
	}
	return sheets, nil
}

func (r *SQLiteRepository) GetSheet(id int64) (model.TestSheet, error) {
	row := r.db.QueryRow(`SELECT id, plan_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, mockup_settings, created_at, updated_at
		FROM test_sheets WHERE id = ?`, id)
	sheet, err := scanSheet(row)
	if err != nil {
		return model.TestSheet{}, err
	}
	sheet.Steps, err = r.ListSteps(id)
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
	res, err := r.db.Exec(`DELETE FROM test_sheets WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
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
		steps = append(steps, step)
	}
	return steps, rows.Err()
}

func (r *SQLiteRepository) GetStep(id int64) (model.TestSheetStep, error) {
	row := r.db.QueryRow(`SELECT id, sheet_id, action, field, expected_result, execution_order, created_at, updated_at
		FROM test_sheet_steps WHERE id = ?`, id)
	return scanStep(row)
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
	res, err := r.db.Exec(`DELETE FROM test_sheet_steps WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if changed, _ := res.RowsAffected(); changed == 0 {
		return sql.ErrNoRows
	}
	return nil
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
	return tx.Commit()
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
	res, err := tx.Exec(`INSERT INTO test_runs (plan_id, plan_name, status, started_at) VALUES (?, ?, ?, ?)`, planID, planName, model.TestRunStatusRunning, now)
	if err != nil {
		return model.TestRun{}, err
	}
	runID, err := res.LastInsertId()
	if err != nil {
		return model.TestRun{}, err
	}
	rows, err := tx.Query(`SELECT id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order
		FROM test_sheets WHERE plan_id = ? ORDER BY execution_order ASC, id ASC`, planID)
	if err != nil {
		return model.TestRun{}, err
	}
	sheets := []model.TestSheet{}
	for rows.Next() {
		var sheet model.TestSheet
		if err := rows.Scan(&sheet.ID, &sheet.Name, &sheet.Description, &sheet.Prerequisites, &sheet.Config, &sheet.Command, &sheet.Notes, &sheet.Action, &sheet.ExpectedResult, &sheet.ExecutionOrder); err != nil {
			_ = rows.Close()
			return model.TestRun{}, err
		}
		sheets = append(sheets, sheet)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return model.TestRun{}, err
	}
	_ = rows.Close()
	for _, sheet := range sheets {
		res, err := tx.Exec(`INSERT INTO test_run_sheets
			(run_id, source_sheet_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, status, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			runID, sheet.ID, sheet.Name, sheet.Description, sheet.Prerequisites, sheet.Config, sheet.Command, sheet.Notes, sheet.Action, sheet.ExpectedResult, sheet.ExecutionOrder, model.RunSheetStatusPending, now, now)
		if err != nil {
			return model.TestRun{}, err
		}
		runSheetID, err := res.LastInsertId()
		if err != nil {
			return model.TestRun{}, err
		}
		stepRows, err := tx.Query(`SELECT id, action, field, expected_result, execution_order FROM test_sheet_steps WHERE sheet_id = ? ORDER BY execution_order ASC, id ASC`, sheet.ID)
		if err != nil {
			return model.TestRun{}, err
		}
		for stepRows.Next() {
			var step model.TestSheetStep
			if err := stepRows.Scan(&step.ID, &step.Action, &step.Field, &step.ExpectedResult, &step.ExecutionOrder); err != nil {
				_ = stepRows.Close()
				return model.TestRun{}, err
			}
			_, err := tx.Exec(`INSERT INTO test_run_steps
				(run_sheet_id, source_step_id, action, field, expected_result, execution_order, status, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				runSheetID, step.ID, step.Action, step.Field, step.ExpectedResult, step.ExecutionOrder, model.RunSheetStatusPending, now, now)
			if err != nil {
				_ = stepRows.Close()
				return model.TestRun{}, err
			}
		}
		if err := stepRows.Err(); err != nil {
			_ = stepRows.Close()
			return model.TestRun{}, err
		}
		_ = stepRows.Close()
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

func (r *SQLiteRepository) ListPlanRuns(planID int64) ([]model.TestRunSummary, error) {
	return r.listRunSummaries(`WHERE r.plan_id = ?`, planID)
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
		summary := model.TestPlanSummary{
			ID:          plan.ID,
			Name:        plan.Name,
			Description: plan.Description,
			Status:      model.TestRunStatusPending,
			SheetCount:  sheetCount,
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
	query := `SELECT r.id, r.plan_id, r.plan_name, r.status, r.started_at, r.finished_at,
		COUNT(DISTINCT rs.id) AS total_sheets,
		COUNT(rst.id) AS total_steps,
		COALESCE(SUM(CASE WHEN rst.status = 'pending' THEN 1 ELSE 0 END), 0) AS pending_steps,
		COALESCE(SUM(CASE WHEN rst.status = 'passed' THEN 1 ELSE 0 END), 0) AS passed_steps,
		COALESCE(SUM(CASE WHEN rst.status = 'failed' THEN 1 ELSE 0 END), 0) AS failed_steps,
		COALESCE(SUM(CASE WHEN rst.status = 'blocked' THEN 1 ELSE 0 END), 0) AS blocked_steps,
		COALESCE(SUM(CASE WHEN rst.status = 'skipped' THEN 1 ELSE 0 END), 0) AS skipped_steps
		FROM test_runs r
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
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

func (r *SQLiteRepository) ReplayRun(runID int64) (model.TestRun, error) {
	source, err := r.GetRun(runID)
	if err != nil {
		return model.TestRun{}, err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return model.TestRun{}, err
	}
	defer rollback(tx)
	now := time.Now().UTC()
	res, err := tx.Exec(`INSERT INTO test_runs (plan_id, plan_name, status, started_at) VALUES (?, ?, ?, ?)`,
		source.PlanID, source.PlanName, model.TestRunStatusRunning, now)
	if err != nil {
		return model.TestRun{}, err
	}
	newRunID, err := res.LastInsertId()
	if err != nil {
		return model.TestRun{}, err
	}
	for _, sheet := range source.Sheets {
		res, err := tx.Exec(`INSERT INTO test_run_sheets
			(run_id, source_sheet_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, status, actual_result, comment, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', '', ?, ?)`,
			newRunID, sheet.SourceSheetID, sheet.Name, sheet.Description, sheet.Prerequisites, sheet.Config, sheet.Command, sheet.Notes, sheet.Action, sheet.ExpectedResult, sheet.ExecutionOrder, model.RunSheetStatusPending, now, now)
		if err != nil {
			return model.TestRun{}, err
		}
		newRunSheetID, err := res.LastInsertId()
		if err != nil {
			return model.TestRun{}, err
		}
		for _, step := range sheet.Steps {
			_, err := tx.Exec(`INSERT INTO test_run_steps
				(run_sheet_id, source_step_id, action, field, expected_result, execution_order, status, actual_result, comment, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, '', '', ?, ?)`,
				newRunSheetID, step.SourceStepID, step.Action, step.Field, step.ExpectedResult, step.ExecutionOrder, model.RunSheetStatusPending, now, now)
			if err != nil {
				return model.TestRun{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return model.TestRun{}, err
	}
	return r.GetRun(newRunID)
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
	rows, err := r.db.Query(`SELECT id, run_id, source_sheet_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, status, actual_result, comment, created_at, updated_at
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
	row := r.db.QueryRow(`SELECT id, run_id, source_sheet_id, name, description, prerequisites, config, command, notes, action, expected_result, execution_order, status, actual_result, comment, created_at, updated_at
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

func scanPlan(scanner interface{ Scan(...any) error }) (model.TestPlan, error) {
	var plan model.TestPlan
	var deleted sql.NullTime
	err := scanner.Scan(&plan.ID, &plan.Name, &plan.Description, &plan.MockupSettings, &plan.CreatedAt, &plan.UpdatedAt, &deleted)
	if deleted.Valid {
		plan.DeletedAt = &deleted.Time
	}
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
	err := scanner.Scan(&sheet.ID, &sheet.PlanID, &sheet.Name, &sheet.Description, &sheet.Prerequisites, &sheet.Config, &sheet.Command, &sheet.Notes, &sheet.Action, &sheet.ExpectedResult, &sheet.ExecutionOrder, &sheet.MockupSettings, &sheet.CreatedAt, &sheet.UpdatedAt)
	return sheet, err
}

func scanStep(scanner interface{ Scan(...any) error }) (model.TestSheetStep, error) {
	var step model.TestSheetStep
	err := scanner.Scan(&step.ID, &step.SheetID, &step.Action, &step.Field, &step.ExpectedResult, &step.ExecutionOrder, &step.CreatedAt, &step.UpdatedAt)
	return step, err
}

func scanRun(scanner interface{ Scan(...any) error }) (model.TestRun, error) {
	var run model.TestRun
	var finished sql.NullTime
	err := scanner.Scan(&run.ID, &run.PlanID, &run.PlanName, &run.Status, &run.StartedAt, &finished)
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
		&summary.PlanID,
		&summary.PlanName,
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

func normalizeRunStatus(status string) string {
	if status == "finished" {
		return model.TestRunStatusCompleted
	}
	return status
}

func scanRunSheet(scanner interface{ Scan(...any) error }) (model.RunSheet, error) {
	var sheet model.RunSheet
	var sourceSheetID sql.NullInt64
	err := scanner.Scan(&sheet.ID, &sheet.RunID, &sourceSheetID, &sheet.Name, &sheet.Description, &sheet.Prerequisites, &sheet.Config, &sheet.Command, &sheet.Notes, &sheet.Action, &sheet.ExpectedResult, &sheet.ExecutionOrder, &sheet.Status, &sheet.ActualResult, &sheet.Comment, &sheet.CreatedAt, &sheet.UpdatedAt)
	if sourceSheetID.Valid {
		value := sourceSheetID.Int64
		sheet.SourceSheetID = &value
	}
	return sheet, err
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
	FOREIGN KEY (source_sheet_id) REFERENCES test_sheets(id) ON DELETE SET NULL
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
	comment TEXT NOT NULL DEFAULT '',
	created_at DATETIME NOT NULL,
	FOREIGN KEY (run_sheet_id) REFERENCES test_run_sheets(id) ON DELETE CASCADE
);
`
