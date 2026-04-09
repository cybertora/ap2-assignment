CREATE TABLE IF NOT EXISTS payments (
                                        id              VARCHAR(36) PRIMARY KEY,
    order_id        VARCHAR(36) NOT NULL,
    transaction_id  VARCHAR(36) NOT NULL UNIQUE,
    amount          BIGINT      NOT NULL CHECK (amount > 0),
    status          VARCHAR(20) NOT NULL DEFAULT 'Authorized',
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
    );

CREATE INDEX idx_payments_order_id ON payments (order_id);
CREATE INDEX idx_payments_transaction_id ON payments (transaction_id);