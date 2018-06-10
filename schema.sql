CREATE TABLE props (
    id bigserial PRIMARY KEY,
    theme int DEFAULT 0 NOT NULL,

    source_author  character varying(64) NOT NULL,
    source_timestamp character varying(64) NOT NULL,
    source_message TEXT NOT NULL,
    source_channel character varying(64) NOT NULL,

    target_timestamp character varying(64) DEFAULT NULL,
    target_channel character varying(64) NOT NULL,

    removed boolean DEFAULT FALSE NOT NULL,

    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

CREATE TABLE reactions (
    id bigserial PRIMARY KEY,

    channel character varying(64) DEFAULT NULL,
    message_timestamp character varying(64) NOT NULL,
    reaction_user character varying(64) NOT NULL,
    reaction character varying(64) NOT NULL,

    removed boolean DEFAULT FALSE NOT NULL,

    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,

    UNIQUE(channel, message_timestamp, reaction_user, reaction)
);