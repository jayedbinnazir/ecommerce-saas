package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/jayedbinnazir/golang-saas.git/internal/permission/domain"
	"github.com/jayedbinnazir/golang-saas.git/internal/platform"
)

type GrantRepository struct {
	db platform.DBTX
}

func NewGrantRepository(db platform.DBTX) *GrantRepository {
	return &GrantRepository{db: db}
}

var _ domain.GrantRepository = (*GrantRepository)(nil)

func (r *GrantRepository) assign(ctx context.Context, table, ownerCol string, ownerID, permissionID uuid.UUID) error {
	q := `INSERT INTO ` + table + ` (` + ownerCol + `, permission_id) VALUES ($1, $2)
	      ON CONFLICT (` + ownerCol + `, permission_id) DO NOTHING`
	res, err := r.db.ExecContext(ctx, q, ownerID, permissionID)
	if err != nil {
		if platform.IsForeignKeyViolation(err) {
			return domain.ErrGrantReferenceInvalid
		}
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrGrantAlreadyExists
	}
	return nil
}

func (r *GrantRepository) revoke(ctx context.Context, table, ownerCol string, ownerID, permissionID uuid.UUID) error {
	q := `DELETE FROM ` + table + ` WHERE ` + ownerCol + ` = $1 AND permission_id = $2`
	res, err := r.db.ExecContext(ctx, q, ownerID, permissionID)
	if err != nil {
		return err
	}
	return platform.AffectedOrNotFound(res, domain.ErrGrantNotFound)
}

func (r *GrantRepository) listOwned(ctx context.Context, joinTable, ownerCol string, ownerID uuid.UUID) ([]domain.Permission, error) {
	q := `SELECT p.id, p.name, p.description, p.created_at, p.updated_at
	      FROM permissions p
	      JOIN ` + joinTable + ` j ON j.permission_id = p.id
	      WHERE j.` + ownerCol + ` = $1
	      ORDER BY p.name`
	rows, err := r.db.QueryContext(ctx, q, ownerID)
	if err != nil {
		return nil, err
	}
	return collectPermissions(rows)
}

func (r *GrantRepository) AssignToRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return r.assign(ctx, "role_permissions", "role_id", roleID, permissionID)
}

func (r *GrantRepository) RevokeFromRole(ctx context.Context, roleID, permissionID uuid.UUID) error {
	return r.revoke(ctx, "role_permissions", "role_id", roleID, permissionID)
}

func (r *GrantRepository) ListByRole(ctx context.Context, roleID uuid.UUID) ([]domain.Permission, error) {
	return r.listOwned(ctx, "role_permissions", "role_id", roleID)
}

func (r *GrantRepository) AssignToMembership(ctx context.Context, membershipID, permissionID uuid.UUID) error {
	return r.assign(ctx, "membership_permissions", "membership_id", membershipID, permissionID)
}

func (r *GrantRepository) RevokeFromMembership(ctx context.Context, membershipID, permissionID uuid.UUID) error {
	return r.revoke(ctx, "membership_permissions", "membership_id", membershipID, permissionID)
}

func (r *GrantRepository) ListByMembership(ctx context.Context, membershipID uuid.UUID) ([]domain.Permission, error) {
	return r.listOwned(ctx, "membership_permissions", "membership_id", membershipID)
}

// EffectiveForMembership unions the role's baseline grants with the member's
// individual grants.
func (r *GrantRepository) EffectiveForMembership(ctx context.Context, membershipID uuid.UUID) ([]string, error) {
	const q = `
		SELECT DISTINCT p.name
		FROM permissions p
		WHERE p.id IN (
			SELECT rp.permission_id
			FROM role_permissions rp
			JOIN memberships m ON m.role_id = rp.role_id
			WHERE m.id = $1
			UNION
			SELECT mp.permission_id
			FROM membership_permissions mp
			WHERE mp.membership_id = $1
		)
		ORDER BY p.name`

	rows, err := r.db.QueryContext(ctx, q, membershipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}
