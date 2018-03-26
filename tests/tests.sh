#!/usr/bin/env bash
echo Testing Server Availability
echo "hello" | nc localhost 12345
cat ../logs/exchange.log

echo Conclude test
