-- Sets or clears the user's selected border. borderID = NULL clears
-- the selection (sets the column to NULL). RETURNING gives us the
-- new value so the caller can decide if a re-hydration is needed
-- without a second SELECT.
--
-- Soft-deleted users (deleted_at IS NOT NULL) are excluded — they
-- shouldn't be selectable. The caller (usecase) translates the
-- zero-rows result into domain.ErrNotFound.
UPDATE users
SET selected_border_id = $2
WHERE id = $1
  AND deleted_at IS NULL
RETURNING selected_border_id
