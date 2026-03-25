// Subscription Export Load Test
// Tests V2Ray/Clash/Singbox subscription generation with 50 concurrent users

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');

// Test configuration
export const options = {
  scenarios: {
    subscription_export: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 25 },
        { duration: '5m', target: 50 },
        { duration: '10m', target: 50 },
        { duration: '3m', target: 0 },
      ],
      gracefulStop: '30s',
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests < 500ms
    http_req_failed: ['rate<0.01'],
    errors: ['rate<0.01'],
  },
};

// Base URL
const BASE_URL = 'http://localhost:8080';

// Test subscription tokens (create these in your test database)
// Format: Array of subscription tokens
const subscriptionTokens = [
  'token1',
  'token2',
  'token3',
  // Add more test tokens as needed
];

// Main test function
export default function () {
  // Select random token
  const token = subscriptionTokens[Math.floor(Math.random() * subscriptionTokens.length)];
  
  if (!token) {
    errorRate.add(1);
    console.error('No subscription token available');
    return;
  }

  // Test 1: V2Ray subscription (base64)
  const v2rayRes = http.get(`${BASE_URL}/sub/${token}`);
  errorRate.add(v2rayRes.status !== 200);
  
  check(v2rayRes, {
    'v2ray subscription status is 200': (r) => r.status === 200,
    'v2ray response time < 500ms': (r) => r.timings.duration < 500,
    'v2ray content-type is correct': (r) => r.headers['Content-Type'].includes('text/plain'),
  });

  sleep(0.5);

  // Test 2: Clash subscription (YAML)
  const clashRes = http.get(`${BASE_URL}/sub/${token}/clash`);
  errorRate.add(clashRes.status !== 200);
  
  check(clashRes, {
    'clash subscription status is 200': (r) => r.status === 200,
    'clash response time < 500ms': (r) => r.timings.duration < 500,
    'clash content-type is correct': (r) => r.headers['Content-Type'].includes('text/yaml') || r.headers['Content-Type'].includes('text/plain'),
  });

  sleep(0.5);

  // Test 3: Sing-box subscription (JSON)
  const singboxRes = http.get(`${BASE_URL}/sub/${token}/singbox`);
  errorRate.add(singboxRes.status !== 200);
  
  check(singboxRes, {
    'singbox subscription status is 200': (r) => r.status === 200,
    'singbox response time < 500ms': (r) => r.timings.duration < 500,
    'singbox content-type is correct': (r) => r.headers['Content-Type'].includes('application/json'),
  });

  sleep(1);
}

// Setup: Verify test data exists
export function setup() {
  console.log(`Starting subscription export load test with ${subscriptionTokens.length} tokens`);
  
  // Test one token to ensure it works
  if (subscriptionTokens.length > 0) {
    const testRes = http.get(`${BASE_URL}/sub/${subscriptionTokens[0]}`);
    if (testRes.status !== 200) {
      console.error('Warning: Test subscription token may not be valid');
      console.error(`Response: ${testRes.body}`);
    }
  }
  
  return {};
}
