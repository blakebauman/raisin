import { betterAuth } from "better-auth";
import { Pool } from "pg";

const connectionString =
  process.env.DATABASE_URL ??
  "postgres://raisin:raisin@localhost:5433/raisin?sslmode=disable";

export const auth = betterAuth({
  database: new Pool({ connectionString }),
  emailAndPassword: {
    enabled: true,
    requireEmailVerification: false,
  },
  trustedOrigins: [
    process.env.BETTER_AUTH_URL ?? "http://localhost:3001",
    "http://localhost:3001",
    "http://localhost:3000",
  ],
});

export type Session = typeof auth.$Infer.Session;
