CREATE DATABASE exchange OWNER andrewbihl;
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
    uid varchar PRIMARY KEY,
    account_id varchar,
    symbol varchar,
    limit_f float,
    amount float
);
CREATE TABLE IF NOT EXISTS sell_order (
    uid varchar PRIMARY KEY,
    account_id varchar,
    symbol varchar,
    limit_f float,
    amount float
);
CREATE TABLE IF NOT EXISTS symbol (
    name varchar PRIMARY KEY,
    shares float
);
CREATE TABLE IF NOT EXISTS transaction (
    symbol varchar,
    amount float,
    price float,
    transaction_time time
);
CREATE INDEX buy_limit ON buy_order (limit_f);
CREATE INDEX sell_limit ON sell_order (limit_f);
