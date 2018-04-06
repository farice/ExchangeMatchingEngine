#!/usr/bin/env bash
echo Testing Server Availability
echo "hello" | nc $1 12345
cat ../logs/exchange.log

echo Testing Sample Create
cat create/sample.txt | nc $1 12345
cat ../logs/exchange.log

echo Stress test with Sample Request
seq $2 | parallel -n0 "cat create/sample.txt | nc $1 12345"
cat ../logs/exchange.log

echo Testing Sample Transactions
cat transaction/buy/1.txt | nc $1 12345 && cat transaction/sell/1.txt | nc $1 12345
cat ../logs/exchange.log

echo Stress test with Sample Transactions
seq $2 | parallel -n0 "cat transaction/buy/1.txt | nc $1 12345 && cat transaction/sell/1.txt | nc $1 12345"
cat ../logs/exchange.log

echo Testing Sample Create + Transactions
cat create/sample.txt | nc $1 12345 && cat transaction/sell/1.txt | nc $1 12345 && cat transaction/buy/1.txt | nc $1 12345
cat ../logs/exchange.log

echo Stress test with Create/Transactions
seq $2 | parallel -n0 "cat create/sample.txt | nc $1 12345 && cat transaction/sell/1.txt | nc $1 12345 && cat transaction/buy/1.txt | nc $1 12345"
cat ../logs/exchange.log

echo Conclude test
