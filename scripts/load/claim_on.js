import http from 'k6/http'
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js'

export const options = {
  scenarios: {
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: 500,
      timeUnit: '1s',
      duration: '1m',
      preAllocatedVUs: 50,
      maxVUs: 500,
    },
  },
}

const text100kb = randomString(100 * 1024)
export default function () {
  const data = { context_text: text100kb, file: http.file(new ArrayBuffer(1), 'a.txt') }
  http.post('http://localhost:8080/v1/ingest', data, { headers: { 'X-Tenant-ID': 'test-1' } })
}
