import http from "k6/http";
import { Rate, Counter } from "k6/metrics";

const realErrors = new Rate("real_errors");
const rpsCounter = new Counter("requests_total");

export let options = {
  stages: [
    // { duration: '10s', target: 500 },   // Плавный старт
    // { duration: '10s', target: 1000 },  // Рост до пика
    // { duration: '10s', target: 1000 },  // Удерживаем пик
    // { duration: '10s', target: 0 },     // Плавное завершение

    { duration: "10s", target: 1000 }, // Удерживаем пик
  ],
  thresholds: {
    real_errors: ["rate<0.01"],
    http_req_duration: ["p(95)<500", "p(99)<1000"],
    http_reqs: ["rate>1500"], // Минимальный RPS
    requests_total: ["rate>1500"], // Кастомный счетчик RPS
  },

  noConnectionReuse: false, // Используем keep-alive
  maxRedirects: 0,
  discardResponseBodies: false, // Оставляем тела ответов для checkout

  setupTimeout: "30s",
  teardownTimeout: "10s",

  // insecureSkipTLSVerify: true,

  userAgent: "k6-load-test/1.0",
};

const BASE_URL = "http://localhost:8000";

const USER_POOL_SIZE = 2000;
const ITEM_POOL_SIZE = 10000;

let userIds, itemIds;

export function setup() {
  userIds = Array.from({ length: USER_POOL_SIZE }, (_, i) => i + 1);
  itemIds = Array.from({ length: ITEM_POOL_SIZE }, (_, i) => i + 1);

  console.log("Setup completed: pools generated");
  return { userIds, itemIds };
}

export default function (data) {
  const userId = data.userIds[Math.floor(Math.random() * data.userIds.length)];
  const itemId = data.itemIds[Math.floor(Math.random() * data.itemIds.length)];

  const params = {
    tags: {
      name: "checkout",
      url: `${BASE_URL}/checkout`,
    },
    timeout: "5s",
    redirects: 0,
    headers: {
      // Connection: "keep-alive",
      "Accept-Encoding": "gzip, deflate",
    },
  };

  const checkoutRes = http.post(
    `${BASE_URL}/checkout?user_id=${userId}&item_id=${itemId}`,
    null,
    params,
  );

  rpsCounter.add(1);

  const isSuccess = checkoutRes.status === 200 || checkoutRes.status === 409;

  if (isSuccess) {
    realErrors.add(0);
  } else {
    realErrors.add(1);
    if (checkoutRes.status >= 500) {
      console.error(`Critical error: ${checkoutRes.status}`);
    }
  }

  if (checkoutRes.status === 409) return;
}

export function teardown(data) {
  console.log("Test completed successfully");
  console.log(`Total test duration: ${__ENV.K6_DURATION || "N/A"}`);
}

// Расширенный summary с RPS
export function handleSummary(data) {
  const totalRequests = data.metrics.http_reqs.values.count;
  const totalDuration = data.state.testRunDurationMs / 1000;
  const avgRPS = Math.round(totalRequests / totalDuration);

  console.log(`\n=== PERFORMANCE SUMMARY ===`);
  console.log(`Total Requests: ${totalRequests}`);
  console.log(`Test Duration: ${totalDuration.toFixed(1)}s`);
  console.log(`Average RPS: ${avgRPS}`);
  console.log(`Max RPS: ${Math.round(data.metrics.http_reqs.values.rate)}`);
  console.log(
    `P95 Response Time: ${data.metrics.http_req_duration.values["p(95)"]}ms`,
  );
  console.log(
    `P99 Response Time: ${data.metrics.http_req_duration.values["p(99)"]}ms`,
  );
  console.log(
    `Error Rate: ${(data.metrics.real_errors?.values.rate * 100 || 0).toFixed(2)}%`,
  );

  return {
    stdout: `
╔══════════════════════════════════════╗
║           RPS RESULTS                ║
╠══════════════════════════════════════╣
║ Average RPS: ${avgRPS.toString().padStart(8)} req/s      ║
║ Total Requests: ${totalRequests.toString().padStart(13)}      ║
║ Duration: ${totalDuration.toFixed(1).padStart(8)}s           ║
╚══════════════════════════════════════╝
        `,
    "summary.json": JSON.stringify(
      {
        ...data,
        custom_metrics: {
          avg_rps: avgRPS,
          total_requests: totalRequests,
          duration_seconds: totalDuration,
        },
      },
      null,
      2,
    ),
  };
}
