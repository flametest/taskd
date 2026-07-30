// taskd backend address per environment. The proxy route (src/app/api/proxy)
// reads this to forward browser requests to the taskd server, avoiding CORS
// and keeping the backend URL server-side.
type TargetEnv = "prod" | "uat" | "dev";

const getTargetEnv = (): TargetEnv => {
  const { TARGET_ENV } = process.env;
  if (TARGET_ENV === "prod") return "prod";
  if (TARGET_ENV === "uat") return "uat";
  return "dev";
};

const taskdBase: Record<TargetEnv, string> = {
  dev: process.env.TASKD_API_BASE ?? "http://localhost:8080",
  uat: process.env.TASKD_API_BASE ?? "http://taskd:8080",
  prod: process.env.TASKD_API_BASE ?? "http://taskd:8080",
};

export { getTargetEnv, taskdBase };
