CREATE TABLE IF NOT EXISTS account (
    uid varchar PRIMARY KEY,
    balance float
);
CREATE TABLE IF NOT EXISTS position (
    uid varchar PRIMARY KEY,
    account_id varchar,
    symbol varchar,
    amount float
);
CREATE TABLE IF NOT EXISTS buy_order (
    uid varchar PRIMARY KEY,
    account_id varchar,
    symbol varchar,
    limit float,
    amount float
);
CREATE TABLE IF NOT EXISTS sell_order (
    uid varchar PRIMARY KEY,
    account_id varchar,
    symbol varchar,
    limit float,
    amount float
);
CREATE TABLE IF NOT EXISTS symbol (
    name varchar PRIMARY KEY,
    shares float
);