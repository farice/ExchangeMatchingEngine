CREATE DATABASE exchange OWNER postgres;
\connect exchange;
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
    price_limit float,
    amount float
);
CREATE TABLE IF NOT EXISTS sell_order (
    uid varchar PRIMARY KEY,
    account_id varchar,
    symbol varchar,
    price_limit float,
    amount float
);
CREATE TABLE IF NOT EXISTS symbol (
    name varchar PRIMARY KEY
);
CREATE TABLE IF NOT EXISTS transaction (
    uid varchar PRIMARY KEY,
    symbol varchar,
    amount float,
    price float,
    transaction_time varchar
);
CREATE INDEX buy_limit ON buy_order (price_limit);
CREATE INDEX sell_limit ON sell_order (price_limit);
