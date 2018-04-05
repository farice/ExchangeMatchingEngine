CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS account (
    uid varchar PRIMARY KEY,
    balance float
);
CREATE TABLE IF NOT EXISTS position (
    account_id varchar,
    symbol varchar,
    amount float, 
    PRIMARY KEY(account_id, symbol)
);
CREATE TABLE IF NOT EXISTS buy_order (
    uid uuid PRIMARY KEY DEFAULT uuid_generate_v1(),
    account_id varchar,
    symbol varchar,
    limit float,
    amount float
);
CREATE TABLE IF NOT EXISTS sell_order (
    uid uuid PRIMARY KEY DEFAULT uuid_generate_v1(),
    account_id varchar,
    symbol varchar,
    limit float,
    amount float
);
CREATE TABLE IF NOT EXISTS symbol (
    name varchar PRIMARY KEY,
    shares float
);