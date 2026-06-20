-- +goose Up
-- +goose StatementBegin
INSERT INTO members (id, name, phone, birth_date, address)
VALUES ('11111111-1111-1111-1111-111111111111', 'Administrador Reinoplus', '11999999999', '1990-01-01', 'Sede da Igreja');

INSERT INTO users (id, name, email, password_hash, role, member_id)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    'Administrador Reinoplus',
    'admin@reinoplus.local',
    '$2a$10$K/5AnEto7rJl7nTe68j0wuTlMn6Led/RI7Hdj8LOxu0vzdisHhqo6',
    'admin',
    '11111111-1111-1111-1111-111111111111'
);
-- password: admin123
-- +goose StatementEnd

-- +goose Down
DELETE FROM users WHERE id = '22222222-2222-2222-2222-222222222222';
DELETE FROM members WHERE id = '11111111-1111-1111-1111-111111111111';
