// Stress Test
// Finds the breaking point by ramping up to 500 concurrent users

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const responseTime = new Trend('response_time');

// Test configuration
export const options = {
  scenarios: {
    stress_test: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5m', target: 50 },   // Ramp to 50
        { duration: '5m', target: 100 },  // Ramp to 100
        { duration: '5m', target: 200 },  // Ramp to 200
        { duration: '5m', target: 500 },  // Ramp to 500 (stress)
        { duration: '10m', target: 500 }, // Hold at 500
        { duration: '5m', target: 0 },    // Ramp down
      ],
      gracefulStop: '30s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'], // More lenient for stress test
    http_req_failed: ['rate<0.1'],    // Allow up to 10% errors under stress
  },
};

// Base URL
const BASE_URL = 'http://localhost:8080';

// Test credentials
const testCredentials = {
  username: 'testadmin',
  password: 'TestPass123!',
};

let authToken = '';

// Setup: Login
export function setup() {
  const loginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
    username: testCredentials.username,
    password: testCredentials.password,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  if (loginRes.status === 200) {
    authToken = loginRes.json('access_token');
  } else {
    console.error('Setup failed: Could not login');
  }

  return { authToken };
}

// Main test function
export default function (data) {
  const token = data.authToken;
  
  if (!token) {
    errorRate.add(1);
    return;
  }

  const headers = {
    'Authorization': `Bearer ${token}`,
    'Content-Type': 'application/json',
  };

  const startTime = Date.now();

  // Mix of read and write operations
  const operations = [
    () => http.get(`${BASE_URL}/api/users?page=1&pageSize=10`, { headers }),
    () => http.get(`${BASE_URL}/api/inbounds`, { headers }),
    () => http.get(`${BASE_URL}/api/cores`, { headers }),
    () => http.get(`${BASE_URL}/api/stats`, { headers }),
    () => http.get(`${BASE_URL}/api/settings`, { headers }),
  ];

  // Randomly select operation
  const operation = operations[Math.floor(Math.random() * operations.length)];
  const res = operation();
  
  const success = res.status >= 200 && res.status < 300;
  errorRate.add(!success);
  responseTime.add(res.timings.duration);

  check(res, {
    'request successful': (r) => r.status >= 200 && r.status < 300,
    'response time < 1s': (r) => r.timings.duration < 1000,
  });

  // Variable sleep to simulate real user behavior
  sleep(Math.random() * 2 + 0.5);

  // Log if we're seeing high error rates
  if (errorRate.rate > 0.1) {
    console.warn(`High error rate detected: ${(errorRate.rate * 100).toFixed(1)}%`);
  }
}

// Teardown: Logout
export function teardown(data) {
  if (data.authToken) {
    http.post(`${BASE_URL}/api/auth/logout`, null, {
      headers: { 'Authorization': `Bearer ${data.authToken}` },
    });
  }
}

// Handle summary
export function handleSummary(data) {
  return {
    'reports/stress-test-summary.json': JSON.stringify(data, null, 2),
    stdout: textSummary(data, { indent: ' ', enableColors: true }),
  };
}

function textSummary(data, options) {
  const { metrics } = data;
  return `
Stress Test Summary:
  Requests: ${metrics.http_reqs?.values?.count || 0}
  Avg Response Time: ${metrics.http_req_duration?.values?.avg?.toFixed(2) || 0}ms
  P95 Response Time: ${metrics.http_req_duration?.values?.['p(95)']?.toFixed(2) || 0}ms
  Max Response Time: ${metrics.http_req_duration?.values?.max?.toFixed(2) || 0}ms
  Error Rate: ${((metrics.http_req_failed?.values?.rate || 0) * 100).toFixed(2)}%
  Max VUs: ${metrics.vus_max?.values?.value || 0}
`;
}
