#!/bin/bash

# Test GELF UDP message
echo "Testing GELF UDP message..."
echo -n '{"version": "1.1", "host": "test-host", "short_message": "UDP test message", "full_message": "This is a full UDP test message\nwith multiple lines", "timestamp": 1671234567.123, "level": 6, "facility": "test", "_user_id": 9001, "_environment": "test"}' | nc -u -w1 localhost 12201

sleep 2

# Test GELF TCP message
echo "Testing GELF TCP message..."
(echo -n '{"version": "1.1", "host": "test-host", "short_message": "TCP test message", "full_message": "This is a full TCP test message", "timestamp": 1671234568.456, "level": 5, "facility": "test", "_request_id": "abc123", "_service": "test-service"}'; echo -ne '\0') | nc -w1 localhost 12201

sleep 2

# Test compressed GELF message (we'll create a simple one)
echo "Testing basic GELF message with additional fields..."
echo -n '{"version": "1.1", "host": "production-server", "short_message": "Error in application", "level": 3, "_error_code": 500, "_module": "auth", "_stack_trace": "line1\nline2\nline3"}' | nc -u -w1 localhost 12201

echo "GELF test messages sent!"
