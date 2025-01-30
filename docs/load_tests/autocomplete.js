// K6_WEB_DASHBOARD=true  k6 run --summary-trend-stats="med,p(95),p(97),p(98),p(99.9)" autocomplete.js  
import http from "k6/http";
import { sleep, check } from "k6";
export const options = {
  stages: [
    { duration: "1m", target: 2000 }, // ramp up
    { duration: "30s", target: 2000 }, // peak
  ],
};

export default () => {
  const reqBody = {
  "query": "Taman Min",
  };

  const res = http.post(
    "http://localhost:6060/api/autocomplete",
    JSON.stringify(reqBody),
    {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
    }
  );
  check(res, { 200: (r) => r.status === 200 });
  sleep(1);
};
