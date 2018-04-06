#!/usr/bin/env bash
echo Testing Server Availability
echo "hello" | nc exchange-engine.colab.duke.edu 12345

echo Testing Sample Create
cat sample_create.txt | nc exchange-engine.colab.duke.edu 12345

echo Stress test with Sample Request
seq 10 | parallel -n0 "cat sample_create.txt | nc exchange-engine.colab.duke.edu 12345"

echo Testing Sample Transactions
cat sample_buy_1.txt | nc exchange-engine.colab.duke.edu 12345 && cat sample_sell_1.txt | nc exchange-engine.colab.duke.edu 12345

echo Stress test with Sample Transactions
seq 10 | parallel -n0 "cat sample_buy_1.txt | nc exchange-engine.colab.duke.edu 12345 && cat sample_sell_1.txt | nc exchange-engine.colab.duke.edu 12345"

echo Testing Sample Create + Transactions
cat sample_create.txt | nc exchange-engine.colab.duke.edu 12345 && cat sample_sell_1.txt | nc exchange-engine.colab.duke.edu 12345 && cat sample_buy_1.txt | nc exchange-engine.colab.duke.edu 12345

echo Stress test with Create/Transactions
seq 10 | parallel -n0 "cat sample_create.txt | nc exchange-engine.colab.duke.edu 12345 && cat sample_sell_1.txt | nc exchange-engine.colab.duke.edu 12345 && cat sample_buy_1.txt | nc exchange-engine.colab.duke.edu 12345"

echo Conclude test
