-- Better Auth expects camelCase column names (quoted identifiers).
-- Fresh 002 installs that ran without proper quoting folded to lowercase.

DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'user' AND column_name = 'emailverified'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'user' AND column_name = 'emailVerified'
  ) THEN
    ALTER TABLE "user" RENAME COLUMN emailverified TO "emailVerified";
    ALTER TABLE "user" RENAME COLUMN createdat TO "createdAt";
    ALTER TABLE "user" RENAME COLUMN updatedat TO "updatedAt";
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'session' AND column_name = 'expiresat'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'session' AND column_name = 'expiresAt'
  ) THEN
    ALTER TABLE "session" RENAME COLUMN expiresat TO "expiresAt";
    ALTER TABLE "session" RENAME COLUMN createdat TO "createdAt";
    ALTER TABLE "session" RENAME COLUMN updatedat TO "updatedAt";
    ALTER TABLE "session" RENAME COLUMN ipaddress TO "ipAddress";
    ALTER TABLE "session" RENAME COLUMN useragent TO "userAgent";
    ALTER TABLE "session" RENAME COLUMN userid TO "userId";
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'account' AND column_name = 'accountid'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'account' AND column_name = 'accountId'
  ) THEN
    ALTER TABLE "account" RENAME COLUMN accountid TO "accountId";
    ALTER TABLE "account" RENAME COLUMN providerid TO "providerId";
    ALTER TABLE "account" RENAME COLUMN userid TO "userId";
    ALTER TABLE "account" RENAME COLUMN accesstoken TO "accessToken";
    ALTER TABLE "account" RENAME COLUMN refreshtoken TO "refreshToken";
    ALTER TABLE "account" RENAME COLUMN idtoken TO "idToken";
    ALTER TABLE "account" RENAME COLUMN accesstokenexpiresat TO "accessTokenExpiresAt";
    ALTER TABLE "account" RENAME COLUMN refreshtokenexpiresat TO "refreshTokenExpiresAt";
    ALTER TABLE "account" RENAME COLUMN createdat TO "createdAt";
    ALTER TABLE "account" RENAME COLUMN updatedat TO "updatedAt";
  END IF;

  IF EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'verification' AND column_name = 'expiresat'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'verification' AND column_name = 'expiresAt'
  ) THEN
    ALTER TABLE "verification" RENAME COLUMN expiresat TO "expiresAt";
    ALTER TABLE "verification" RENAME COLUMN createdat TO "createdAt";
    ALTER TABLE "verification" RENAME COLUMN updatedat TO "updatedAt";
  END IF;
END $$;
