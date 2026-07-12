-- +goose Down
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS org_member_role;
DROP TYPE IF EXISTS org_type;
