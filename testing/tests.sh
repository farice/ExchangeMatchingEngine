#!/usr/bin/env bash

rm ../logs/exchange.log
echo Testing Server Availability
echo "hello" | nc localhost 12345
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Testing Sample Create
cat create/sample.txt | nc localhost 12345
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Stress test with Sample Request
seq 10 | parallel -n0 "cat create/sample.txt | nc localhost 12345"
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Testing Sample Buy
cat transaction/buy/1.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Testing Sample Sell
cat transaction/sell/1.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Testing Sample Buy
cat transaction/buy/1.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Testing Sample Cancel
cat transaction/cancel/1.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Testing Sample Query
cat transaction/query/1.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Stress test with Sample Buys
seq 10 | parallel -n0 "cat transaction/buy/1.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345"
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Stress test with Sample Sells
seq 10 | parallel -n0 "cat transaction/buy/1.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345"
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Stress test with Sample Cancels
seq 10 | parallel -n0 "cat transaction/cancel/2.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345"
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Testing Sample Create + Transactions
cat create/sample.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345 && cat transaction/buy/1.txt | nc localhost 12345
cat ../logs/exchange.log

rm ../logs/exchange.log
echo Stress test with Create/Transactions
seq 10 | parallel -n0 "cat create/sample.txt | nc localhost 12345 && cat transaction/sell/1.txt | nc localhost 12345 && cat transaction/buy/1.txt | nc localhost 12345"
cat ../logs/exchange.log

echo Conclude test
