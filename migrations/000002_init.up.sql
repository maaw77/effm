CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    service_name TEXT NOT NULL,
    price INT NOT NULL CHECK (price >= 0),
    user_id UUID NOT NULL,
    year INT NOT NULL CHECK (year >= 2000 AND year <= 2100),
    month INT NOT NULL CHECK (month BETWEEN 1 AND 12),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(user_id, service_name, year, month)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_year_month 
    ON subscriptions(user_id, year, month);

CREATE INDEX IF NOT EXISTS idx_subscriptions_service_year_month 
    ON subscriptions(service_name, year, month);

CREATE INDEX IF NOT EXISTS idx_subscriptions_year_month 
    ON subscriptions(year, month);