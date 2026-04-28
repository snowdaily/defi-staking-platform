-- Indexer state ----------------------------------------------------------
CREATE TABLE indexer_state (
    id              SERIAL PRIMARY KEY,
    contract        TEXT      NOT NULL,
    last_block      BIGINT    NOT NULL,
    last_block_hash BYTEA     NOT NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (contract)
);

-- Block hash trail used to detect reorgs ---------------------------------
CREATE TABLE block_trail (
    block_number  BIGINT      PRIMARY KEY,
    block_hash    BYTEA       NOT NULL,
    parent_hash   BYTEA       NOT NULL,
    seen_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Per-event tables -------------------------------------------------------
CREATE TABLE deposits (
    tx_hash       BYTEA       NOT NULL,
    log_index     INT         NOT NULL,
    block_number  BIGINT      NOT NULL,
    block_hash    BYTEA       NOT NULL,
    sender        BYTEA       NOT NULL,
    owner         BYTEA       NOT NULL,
    assets        NUMERIC(78) NOT NULL,
    shares        NUMERIC(78) NOT NULL,
    timestamp     TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tx_hash, log_index)
);
CREATE INDEX deposits_owner_idx ON deposits(owner);
CREATE INDEX deposits_block_idx ON deposits(block_number);

CREATE TABLE withdrawals (
    tx_hash       BYTEA       NOT NULL,
    log_index     INT         NOT NULL,
    block_number  BIGINT      NOT NULL,
    block_hash    BYTEA       NOT NULL,
    sender        BYTEA       NOT NULL,
    receiver      BYTEA       NOT NULL,
    owner         BYTEA       NOT NULL,
    assets        NUMERIC(78) NOT NULL,
    shares        NUMERIC(78) NOT NULL,
    timestamp     TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tx_hash, log_index)
);
CREATE INDEX withdrawals_owner_idx ON withdrawals(owner);
CREATE INDEX withdrawals_block_idx ON withdrawals(block_number);

CREATE TABLE reward_distributions (
    tx_hash             BYTEA       NOT NULL,
    log_index           INT         NOT NULL,
    block_number        BIGINT      NOT NULL,
    block_hash          BYTEA       NOT NULL,
    operator            BYTEA       NOT NULL,
    amount              NUMERIC(78) NOT NULL,
    total_assets_after  NUMERIC(78) NOT NULL,
    timestamp           TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (tx_hash, log_index)
);
CREATE INDEX rewards_block_idx ON reward_distributions(block_number);

-- Daily exchange-rate snapshots, used for APR calculation ----------------
CREATE TABLE exchange_rate_snapshots (
    id            SERIAL PRIMARY KEY,
    block_number  BIGINT      NOT NULL,
    timestamp     TIMESTAMPTZ NOT NULL,
    total_assets  NUMERIC(78) NOT NULL,
    total_supply  NUMERIC(78) NOT NULL,
    rate_e27      NUMERIC(78) NOT NULL  -- assets-per-share scaled by 1e27
);
CREATE INDEX rate_ts_idx ON exchange_rate_snapshots(timestamp);
