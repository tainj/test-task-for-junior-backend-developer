BEGIN;

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS recurrence_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS recurrence_config JSONB,
    ADD COLUMN IF NOT EXISTS next_run_date TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS parent_task_id BIGINT REFERENCES tasks(id) ON DELETE CASCADE;

-- Частичные индексы: быстрые и лёгкие, так как заполнены только у периодических задач
CREATE INDEX IF NOT EXISTS idx_tasks_next_run ON tasks(next_run_date) WHERE next_run_date IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_task_id) WHERE parent_task_id IS NOT NULL;

COMMIT;