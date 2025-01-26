import http from "k6/http";
import { sleep, check } from "k6";
export const options = {
  stages: [
    { duration: "1m", target: 800 }, // ramp up
    { duration: "30s", target: 800 }, // peak
  ],
};

export default () => {
  const reqBody = {
  "query": "Kebun Binatang Ragunan",
    "top_k": 10
  };

  const res = http.post(
    "http://localhost:6060/api/search",
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
