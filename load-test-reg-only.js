import http from 'k6/http';
import { check } from 'k6';

export const options = {
    vus: 500,
    duration: '60s',
};

export default function () {
    const url = 'http://app:8080/auth/register';
    // Нам нужна ГАРАНТИРОВАННАЯ уникальность, чтобы не валить батчи в БД
    const timestamp = Date.now();
    const rand = Math.floor(Math.random() * 1000000);
    const id = `${timestamp}-${__VU}-${__ITER}-${rand}`;

    const payload = JSON.stringify({
        email: `extreme-${id}@example.com`,
        username: `u-${id}`,
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
}
