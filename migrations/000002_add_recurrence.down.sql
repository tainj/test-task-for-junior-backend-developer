BEGIN;

DROP INDEX IF EXISTS idx_tasks_parent;
DROP INDEX IF EXISTS idx_tasks_next_run;

ALTER TABLE tasks
    DROP COLUMN IF EXISTS parent_task_id,
    DROP COLUMN IF EXISTS next_run_date,
    DROP COLUMN IF EXISTS recurrence_config,
    DROP COLUMN IF EXISTS recurrence_type;

COMMIT;