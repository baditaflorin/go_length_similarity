// loadtest.js - k6 load testing script for similarity server
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate } from 'k6/metrics';

// Custom metrics
const requestCount = new Counter('request_count');
const errorRate = new Rate('error_rate');

// Test configuration
export const options = {
    scenarios: {
        length_similarity: {
            executor: 'ramping-vus',
            startVUs: 1,
            stages: [
                { duration: '30s', target: 50 },  // Ramp up
                { duration: '1m', target: 50 },   // Steady load
                { duration: '30s', target: 100 }, // Ramp up more
                { duration: '1m', target: 100 },  // Steady higher load
                { duration: '30s', target: 0 },   // Ramp down
            ],
            gracefulRampDown: '10s',
        },

        character_similarity: {
            executor: 'ramping-vus',
            startVUs: 1,
            stages: [
                { duration: '30s', target: 40 },
                { duration: '1m', target: 40 },
                { duration: '30s', target: 80 },
                { duration: '1m', target: 80 },
                { duration: '30s', target: 0 },
            ],
            gracefulRampDown: '10s',
            startTime: '3m30s', // Start after length_similarity
        },

        efficient_streaming: {
            executor: 'ramping-vus',
            startVUs: 1,
            stages: [
                { duration: '30s', target: 30 },
                { duration: '1m', target: 30 },
                { duration: '30s', target: 60 },
                { duration: '1m', target: 60 },
                { duration: '30s', target: 0 },
            ],
            gracefulRampDown: '10s',
            startTime: '7m', // Start after character_similarity
        },
    },
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
        error_rate: ['rate<0.01'],        // Error rate should be less than 1%
    },
};

// Test data
const smallPayload = {
    original: "The quick brown fox jumps over the lazy dog.",
    augmented: "The swift brown fox leaps over the sleepy dog.",
    threshold: 0.7
};

const mediumPayload = {
    original: "This is a longer text that will test the performance of the server with a medium-sized payload. " +
        "The server should be able to handle this size of text without significant performance degradation. " +
        "We need to make sure the memory usage remains stable and response times are consistent.",
    augmented: "This is a similar but modified text that will test the performance of the server with a medium-sized payload. " +
        "The service should be capable of processing this amount of text without notable performance issues. " +
        "We want to ensure memory consumption stays efficient and response latency remains predictable.",
    threshold: 0.7
};

const largePayload = {
    original: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. ".repeat(50),
    augmented: "Lorem ipsum dolor sit amet, consectetur adipiscing elit. ".repeat(45) +
        "Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. ".repeat(5),
    threshold: 0.7
};

// Base URL
const baseUrl = __ENV.TARGET_URL || 'http://localhost:8080';

export function setup() {
    // Check if the server is up
    const res = http.get(`${baseUrl}/health`);
    check(res, {
        'server is up': (r) => r.status === 200,
    });

    return {
        baseUrl,
    };
}

export default function(data) {
    // Determine which endpoint to test based on scenario execution
    let endpoint;
    let payload;

    if (__ITER <= 50) {
        payload = smallPayload;
    } else if (__ITER <= 100) {
        payload = mediumPayload;
    } else {
        payload = largePayload;
    }

    // Select endpoint based on the current scenario
    if (__ENV.SCENARIO === 'length_similarity') {
        endpoint = '/length';
    } else if (__ENV.SCENARIO === 'character_similarity') {
        endpoint = '/character';
    } else {
        endpoint = '/efficient';
    }

    // Make the request
    const res = http.post(
        `${data.baseUrl}${endpoint}`,
        JSON.stringify(payload),
        { headers: { 'Content-Type': 'application/json' } }
    );

    // Count the request
    requestCount.add(1);

    // Check if response is valid
    const success = check(res, {
        'status is 200': (r) => r.status === 200,
        'response has score': (r) => r.json('score') !== undefined,
    });

    // Update error rate if check failed
    if (!success) {
        errorRate.add(1);
        console.log(`Error with ${endpoint}: ${res.status} ${res.body}`);
    }

    // Random sleep between requests
    sleep(Math.random() * 0.5);
}

export function teardown(data) {
    console.log('Load test completed');
}