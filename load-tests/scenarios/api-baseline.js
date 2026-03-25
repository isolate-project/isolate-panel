// API Baseline Load Test
// Tests basic API endpoint performance with 100 concurrent users

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  scenarios: {
    api_baseline: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5m', target: 50 },   // Ramp up to 50 users
        { duration: '10m', target: 100 }, // Ramp up to 100 users
        { duration: '30m', target: 100 }, // Stay at 100 users
        { duration: '5m', target: 0 },    // Ramp down
      ],
      gracefulStop: '30s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<100'], // 95% of requests < 100ms
    http_req_failed: ['rate<0.01'],   // Error rate < 1%
    errors: ['rate<0.01'],            // Custom error rate < 1%
  },
};

// Base URL - change this to your test server
const BASE_URL = 'http://localhost:8080';

// Test data
const testCredentials = {
  username: 'testadmin',
  password: 'TestPass123!',
};

let authToken = '';

// Setup: Login and get auth token
export function setup() {
  const loginRes = http.post(`${BASE_URL}/api/auth/login`, JSON.stringify({
    username: testCredentials.username,
    password: testCredentials.password,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  const loginSuccess = check(loginRes, {
    'login status is 200': (r) => r.status === 200,
  });

  if (loginSuccess) {
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

  // Test 1: Get users list
  const usersRes = http.get(`${BASE_URL}/api/users?page=1&pageSize=10`, { headers });
  errorRate.add(usersRes.status !== 200);
  
  check(usersRes, {
    'get users status is 200': (r) => r.status === 200,
    'get users response time < 100ms': (r) => r.timings.duration < 100,
  });

  sleep(0.5);

  // Test 2: Get inbounds list
  const inboundsRes = http.get(`${BASE_URL}/api/inbounds`, { headers });
  errorRate.add(inboundsRes.status !== 200);
  
  check(inboundsRes, {
    'get inbounds status is 200': (r) => r.status === 200,
    'get inbounds response time < 100ms': (r) => r.timings.duration < 100,
  });

  sleep(0.5);

  // Test 3: Get cores status
  const coresRes = http.get(`${BASE_URL}/api/cores`, { headers });
  errorRate.add(coresRes.status !== 200);
  
  check(coresRes, {
    'get cores status is 200': (r) => r.status === 200,
    'get cores response time < 100ms': (r) => r.timings.duration < 100,
  });

  sleep(0.5);

  // Test 4: Get settings
  const settingsRes = http.get(`${BASE_URL}/api/settings`, { headers });
  errorRate.add(settingsRes.status !== 200);
  
  check(settingsRes, {
    'get settings status is 200': (r) => r.status === 200,
    'get settings response time < 100ms': (r) => r.timings.duration < 100,
  });

  sleep(0.5);

  // Test 5: Get dashboard stats
  const statsRes = http.get(`${BASE_URL}/api/stats`, { headers });
  errorRate.add(statsRes.status !== 200);
  
  check(statsRes, {
    'get stats status is 200': (r) => r.status === 200,
    'get stats response time < 100ms': (r) => r.timings.duration < 100,
  });

  sleep(1);
}

// Teardown: Logout
export function teardown(data) {
  if (data.authToken) {
    http.post(`${BASE_URL}/api/auth/logout`, null, {
      headers: { 'Authorization': `Bearer ${data.authToken}` },
    });
  }
}
