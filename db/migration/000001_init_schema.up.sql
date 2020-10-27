CREATE TABLE "cells" (
  "id" bigserial PRIMARY KEY,
  "balls" int NOT NULL
);


CREATE INDEX ON "cells" ("balls");
