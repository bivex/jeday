import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 50 }, // ramp up to 50 users
    { duration: '20s', target: 100 }, // stay at 100 users
    { duration: '5s', target: 0 },  // ramp down
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'], // http errors should be less than 1%
    http_req_duration: ['p(95)<200'], // 95% of requests should be below 200ms
  },
};

export default function () {
  const url = 'http://app:8080/health';
  const res = http.get(url);
  
  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  sleep(0.1); // Small sleep to control request rate per VU
}
