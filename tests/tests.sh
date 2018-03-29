#!/usr/bin/env bash
echo Testing Server Availability
echo "hello" | nc localhost 12345
cat ../logs/exchange.log

echo Testing Sample Request
cat sample_request.txt | nc localhost 12345
cat ../logs/exchange.log

echo Stress test with Sample Request
seq 10 | parallel -n0 "cat sample_request.txt | nc localhost 12345"
cat ../logs/exchange.log

echo Testing Sample Transactions
cat sample_transactions.txt | nc localhost 12345
cat ../logs/exchange.log

echo Stress test with Sample Transactions
seq 10 | parallel -n0 "cat sample_transactions.txt | nc localhost 12345"
cat ../logs/exchange.log

echo Conclude test
