// Concurrent Writes Load Test
// Tests database write contention with 20 concurrent users

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';
import { uuid } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  scenarios: {
    concurrent_writes: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 10 },
        { duration: '5m', target: 20 },
        { duration: '10m', target: 20 },
        { duration: '3m', target: 0 },
      ],
      gracefulStop: '30s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<50'],  // 95% of requests < 50ms
    http_req_failed: ['rate<0.01'],
    errors: ['rate<0.01'],
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

  // Test 1: Create setting (write operation)
  const settingKey = `test_setting_${uuid()}`;
  const settingRes = http.put(
    `${BASE_URL}/api/settings/${settingKey}`,
    JSON.stringify({
      key: settingKey,
      value: `test_value_${uuid()}`,
      value_type: 'string',
    }),
    { headers }
  );
  errorRate.add(settingRes.status !== 200 && settingRes.status !== 201);
  
  check(settingRes, {
    'create setting status is 2xx': (r) => r.status >= 200 && r.status < 300,
    'create setting response time < 50ms': (r) => r.timings.duration < 50,
  });

  sleep(0.2);

  // Test 2: Update setting (write operation)
  const updateRes = http.put(
    `${BASE_URL}/api/settings/${settingKey}`,
    JSON.stringify({
      key: settingKey,
      value: `updated_value_${uuid()}`,
      value_type: 'string',
    }),
    { headers }
  );
  errorRate.add(updateRes.status !== 200);
  
  check(updateRes, {
    'update setting status is 200': (r) => r.status === 200,
    'update setting response time < 50ms': (r) => r.timings.duration < 50,
  });

  sleep(0.2);

  // Test 3: Delete setting (write operation)
  const deleteRes = http.del(`${BASE_URL}/api/settings/${settingKey}`, null, { headers });
  errorRate.add(deleteRes.status !== 200 && deleteRes.status !== 204);
  
  check(deleteRes, {
    'delete setting status is 2xx': (r) => r.status >= 200 && r.status < 300,
    'delete setting response time < 50ms': (r) => r.timings.duration < 50,
  });

  sleep(0.5);
}

// Teardown: Logout
export function teardown(data) {
  if (data.authToken) {
    http.post(`${BASE_URL}/api/auth/logout`, null, {
      headers: { 'Authorization': `Bearer ${data.authToken}` },
    });
  }
}
