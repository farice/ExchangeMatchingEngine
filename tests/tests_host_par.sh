#!/usr/bin/env bash
echo Testing Server Availability
echo "hello" | nc $1 12345
cat ../logs/exchange.log

echo Testing Sample Create
cat sample_create.txt | nc $1 12345
cat ../logs/exchange.log

echo Stress test with Sample Request
seq $2 | parallel -n0 "cat sample_create.txt | nc $1 12345"
cat ../logs/exchange.log

echo Testing Sample Transactions
cat sample_buy_1.txt | nc $1 12345 && cat sample_sell_1.txt | nc $1 12345
cat ../logs/exchange.log

echo Stress test with Sample Transactions
seq $2 | parallel -n0 "cat sample_buy_1.txt | nc $1 12345 && cat sample_sell_1.txt | nc $1 12345"
cat ../logs/exchange.log

echo Testing Sample Create + Transactions
cat sample_create.txt | nc $1 12345 && cat sample_sell_1.txt | nc $1 12345 && cat sample_buy_1.txt | nc $1 12345
cat ../logs/exchange.log

echo Stress test with Create/Transactions
seq $2 | parallel -n0 "cat sample_create.txt | nc $1 12345 && cat sample_sell_1.txt | nc $1 12345 && cat sample_buy_1.txt | nc $1 12345"
cat ../logs/exchange.log

echo Conclude test
