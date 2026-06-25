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
