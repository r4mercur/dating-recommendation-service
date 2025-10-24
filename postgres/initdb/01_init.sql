-- postgres
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    interest TEXT[] NOT NULL DEFAULT '{}',
    hobby TEXT[] NOT NULL DEFAULT '{}',
    age INTEGER NOT NULL CHECK (age >= 18 AND age <= 60),
    address TEXT NOT NULL,
    gender VARCHAR(50),
    status VARCHAR(50),
    photo TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_age ON users(age);
CREATE INDEX IF NOT EXISTS idx_users_gender ON users(gender);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
