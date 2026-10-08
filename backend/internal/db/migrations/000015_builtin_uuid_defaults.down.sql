ALTER TABLE users          ALTER COLUMN id SET DEFAULT uuid_generate_v4();
ALTER TABLE refresh_tokens ALTER COLUMN id SET DEFAULT uuid_generate_v4();
ALTER TABLE habits         ALTER COLUMN id SET DEFAULT uuid_generate_v4();
ALTER TABLE habit_logs     ALTER COLUMN id SET DEFAULT uuid_generate_v4();
ALTER TABLE audit_logs     ALTER COLUMN id SET DEFAULT uuid_generate_v4();
