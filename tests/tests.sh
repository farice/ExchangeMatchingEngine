#!/usr/bin/env bash
echo Testing Server Availability
rm ../logs/exchange.log
echo "hello" | nc localhost 12345
cat ../logs/exchange.log

echo Testing Sample Create
rm ../logs/exchange.log
cat sample_create.txt | nc localhost 12345
cat ../logs/exchange.log

echo Stress test with Sample Request
rm ../logs/exchange.log
seq 10 | parallel -n0 "cat sample_create.txt | nc localhost 12345"
cat ../logs/exchange.log

echo Testing Sample Transactions
rm ../logs/exchange.log
cat sample_buy_1.txt | nc localhost 12345 && cat sample_sell_1.txt | nc localhost 12345
cat ../logs/exchange.log

echo Stress test with Sample Transactions
rm ../logs/exchange.log
seq 10 | parallel -n0 "cat sample_buy_1.txt | nc localhost 12345 && cat sample_sell_1.txt | nc localhost 12345"
cat ../logs/exchange.log

echo Testing Sample Create + Transactions
rm ../logs/exchange.log
cat sample_create.txt | nc localhost 12345 && cat sample_sell_1.txt | nc localhost 12345 && cat sample_buy_1.txt | nc localhost 12345
cat ../logs/exchange.log

echo Stress test with Create/Transactions
rm ../logs/exchange.log
seq 10 | parallel -n0 "cat sample_create.txt | nc localhost 12345 && cat sample_sell_1.txt | nc localhost 12345 && cat sample_buy_1.txt | nc localhost 12345"
cat ../logs/exchange.log

echo Conclude test
