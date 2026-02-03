-- name: CreateFormSubmission :one
INSERT INTO form_submission (id, site_id, form_type, name, email, message, ip_address, user_agent, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetFormSubmission :one
SELECT * FROM form_submission WHERE id = ?;

-- name: ListFormSubmissionsBySite :many
SELECT * FROM form_submission WHERE site_id = ? ORDER BY created_at DESC;

-- name: CountUnreadFormSubmissions :one
SELECT COUNT(*) FROM form_submission WHERE site_id = ? AND read_at IS NULL;

-- name: MarkFormSubmissionRead :exec
UPDATE form_submission SET read_at = ? WHERE id = ?;

-- name: DeleteFormSubmission :exec
DELETE FROM form_submission WHERE id = ?;
