import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '5s', target: 10 },
        { duration: '10s', target: 20 },
        { duration: '5s', target: 0 },
    ],
};

export default function () {
    const url = 'http://app:8080/auth/register';
    const id = Math.random().toString(36).substring(7);
    const payload = JSON.stringify({
        email: `user-${id}@example.com`,
        username: `user-${id}`,
        password: 'password123',
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const res = http.post(url, payload, params);

    check(res, {
        'status is 200': (r) => r.status === 200,
    });

    sleep(0.1);
}
