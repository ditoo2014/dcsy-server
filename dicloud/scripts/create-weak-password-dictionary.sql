BEGIN;

CREATE TABLE IF NOT EXISTS public.weak_password_dictionary (
	id bigserial PRIMARY KEY,
	password varchar(255) NOT NULL
);

DROP INDEX IF EXISTS public.idx_weak_password_dictionary_lower;
CREATE INDEX idx_weak_password_dictionary_lower
ON public.weak_password_dictionary (LOWER(password));

COMMIT;
