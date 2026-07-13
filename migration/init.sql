CREATE TABLE task
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version     BIGINT       NOT NULL DEFAULT 0,
    name        VARCHAR(255) NOT NULL,
    ref_id      VARCHAR(64)  NOT NULL,
    protocol    VARCHAR(16)  NOT NULL,
    address     VARCHAR(500) NOT NULL,
    params JSONB,
    exec_time TIMESTAMPTZ NOT NULL,
    status      VARCHAR(16)  NOT NULL,
    attempts    BIGINT       NOT NULL DEFAULT 0,
    max_retries BIGINT       NOT NULL,
    last_error  TEXT,
    locked_until TIMESTAMPTZ,
    cron         VARCHAR(64),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    UNIQUE (ref_id)
);

CREATE INDEX idx_task_created_at ON task (created_at);
CREATE INDEX idx_task_updated_at ON task (updated_at);
CREATE INDEX idx_task_deleted_at ON task (deleted_at);
CREATE INDEX idx_task_status ON task (status);
CREATE INDEX idx_task_exec_time ON task (exec_time);
CREATE INDEX idx_task_claim ON task (status, exec_time);
CREATE INDEX idx_task_locked_until ON task (locked_until);

-- task_record is the append-only execution-audit log: one row per executor.Execute
-- call (success, retryable failure, or non-retryable failure). task : task_record
-- = 1 : N. Recording is best-effort and never coupled to the task state machine.
CREATE TABLE task_record
(
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version       BIGINT        NOT NULL DEFAULT 0,
    task_id       UUID          NOT NULL, -- soft-FK to task.id; no constraint (matches task table style)
    attempt       BIGINT        NOT NULL, -- 1-based index of THIS execution
    result        VARCHAR(16)   NOT NULL, -- 'success' | 'failure'
    protocol      VARCHAR(16)   NOT NULL, -- snapshot of task.protocol at exec time
    instance_id   VARCHAR(128)  NOT NULL, -- which scheduler instance ran it
    error_message TEXT,                    -- NULL on success; the executor's err.Error() on failure
    started_at    TIMESTAMPTZ   NOT NULL,
    finished_at   TIMESTAMPTZ   NOT NULL,
    duration_ms   BIGINT        NOT NULL, -- finished_at - started_at, in ms
    response      JSONB,                    -- reserved; NULL this round (executors don't capture bodies yet)
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    TIMESTAMPTZ
);

-- hot query: list one task's history newest-first
CREATE INDEX idx_task_record_task_id_created_at ON task_record (task_id, created_at DESC);
CREATE INDEX idx_task_record_created_at ON task_record (created_at);
CREATE INDEX idx_task_record_updated_at ON task_record (updated_at);
CREATE INDEX idx_task_record_deleted_at ON task_record (deleted_at);
