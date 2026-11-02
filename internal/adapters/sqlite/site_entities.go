package sqlite

import "context"

const (
	deleteLeafEntities = `DELETE FROM entities WHERE site_id = ? AND NOT EXISTS (
    SELECT 1 FROM entities child WHERE child.scope_entity_id = entities.id)`
	freeLoopedEntities = `UPDATE entities SET scope_entity_id = NULL, name = id WHERE site_id = ?`
	deleteSiteEntities = `DELETE FROM entities WHERE site_id = ?`
)

func (s *Store) deleteEntitiesLeavesFirst(ctx context.Context, siteID string) error {
	exec := s.writeFrom(ctx)
	for {
		affected, err := execWrite(ctx, exec, deleteLeafEntities, []any{siteID}, nil, "delete the site's entities")
		if err != nil {
			return err
		}
		if affected == 0 {
			break
		}
	}
	if _, err := execWrite(ctx, exec, freeLoopedEntities, []any{siteID}, nil, "free the site's looped entities"); err != nil {
		return err
	}
	_, err := execWrite(ctx, exec, deleteSiteEntities, []any{siteID}, nil, "delete the site's looped entities")
	return err
}
