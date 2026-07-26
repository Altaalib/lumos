CREATE TABLE posts (
    id           bigserial PRIMARY KEY,
    channel_id   bigint      NOT NULL,
    message_id   bigint      NOT NULL,
    text         text        NOT NULL,  -- message.Text либо message.Caption
    created_at   timestamptz NOT NULL DEFAULT now(),
    status       text        NOT NULL DEFAULT 'new',
        -- 'new' -> 'analyzed' -> 'sent'
    importance   boolean,     -- null пока не проанализирован,
                               -- true/false — отправлять или нет
    analyzed_at  timestamptz,
    sent_at      timestamptz,
    UNIQUE (channel_id, message_id)
);

CREATE TABLE read_checkpoints (
    channel_id      bigint PRIMARY KEY,
    last_message_id bigint,
    last_read_at    timestamptz
);
