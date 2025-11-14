CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE subscriptions (
                               id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                               service_name TEXT NOT NULL,
                               price INTEGER NOT NULL CHECK (price > 0),
                               user_id UUID NOT NULL,
                               start_date TEXT NOT NULL,
                               end_date TEXT
);