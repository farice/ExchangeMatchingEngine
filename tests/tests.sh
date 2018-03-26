#!/usr/bin/env bash
echo Testing Server Availability
echo "hello" | nc localhost 12345
cat ../logs/exchange.log

echo Testing Sample Request
cat sample_request.txt | nc localhost 12345
cat ../logs/exchange.log

echo Conclude test
