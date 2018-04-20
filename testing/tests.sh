#!/usr/bin/env bash

rm ../logs/exchange.log

echo Testing Sample Create
cat create/sample.txt | nc localhost 12345

echo Stress test with Sample Request
seq 10 | parallel -n0 "cat create/sample.txt | nc localhost 12345"

echo Testing Sample Buy
cat transaction/buy/1.txt | nc localhost 12345 && cat transaction/buy/1.txt | nc localhost 12345

echo Testing Sample Sell
cat transaction/sell/1.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345

echo Testing Sample Match
cat transaction/buy/1.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345

echo Testing Sample Query
cat transaction/query/1.txt | nc localhost 12345

echo Testing Sample Cancel
cat transaction/buy/1.txt | nc localhost 12345 && cat transaction/cancel/1.txt | nc localhost 12345

echo Stress test with Sample Matches
seq 10 | parallel -n0 "cat transaction/buy/1.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345"

echo Stress test with Sample Cancels
seq 10 | parallel -n0 "cat transaction/cancel/2.txt | nc localhost 12345"

echo Testing Sample Create + Transactions
cat create/sample.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345 && cat transaction/buy/1.txt | nc localhost 12345

echo Stress test with Create/Transactions
seq 10 | parallel -n0 "cat create/sample.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345 && cat transaction/buy/1.txt | nc localhost 12345"

echo Conclude test
