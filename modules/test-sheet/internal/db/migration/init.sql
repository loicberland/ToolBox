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
