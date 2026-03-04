import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
    stages: [
        { duration: '10s', target: 50 },
        { duration: '20s', target: 100 },
        { duration: '5s', target: 0 },
    ],
};

export default function () {
    const registerUrl = 'http://app:8080/auth/register';
    const loginUrl = 'http://app:8080/auth/login';
    const meUrl = 'http://app:8080/auth/me';

    const id = Math.random().toString(36).substring(7);
    const email = `user-${id}-${__VU}-${__ITER}@example.com`;
    const password = 'password123';

    // 1. Register (FAST PATH)
    const regPayload = JSON.stringify({
        email: email,
        username: `user-${id}`,
        password: password,
    });
    const params = { headers: { 'Content-Type': 'application/json' } };
    const regRes = http.post(registerUrl, regPayload, params);
    check(regRes, { 'reg status is 200': (r) => r.status === 200 });

    // 2. Login
    const loginPayload = JSON.stringify({ email: email, password: password });
    const loginRes = http.post(loginUrl, loginPayload, params);
    check(loginRes, { 'login status is 200': (r) => r.status === 200 });

    if (loginRes.status === 200) {
        const token = loginRes.json('access_token');
        const authParams = {
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${token}`,
            },
        };

        // 3. Get /me
        const meRes = http.get(meUrl, authParams);
        check(meRes, { 'me status is 200': (r) => r.status === 200 });
    }

    sleep(0.05); // Faster iteration
}
