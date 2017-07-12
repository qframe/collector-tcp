#!/bin/bash

echo 'cee{"data": "test 123", "event_code": "001.001"}' | nc -w1  $1 11001
