package sqlite

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

const (
	siteColumns = `id, name, base_url, username, secret_ref, status, allow_insecure, plugin_installed, plugin_version, plugin_capabilities, plugin_seo, default_template_id, default_link_policy_id, model_profiles, created_at, updated_at`
	insertSite  = `INSERT INTO sites (` + siteColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	updateSite  = `UPDATE sites SET name = ?, base_url = ?, username = ?, status = ?, allow_insecure = ?, plugin_installed = ?, plugin_version = ?, plugin_capabilities = ?, plugin_seo = ?, default_template_id = ?, default_link_policy_id = ?, model_profiles = ?, updated_at = ? WHERE id = ?`
	deleteSite  = `DELETE FROM sites WHERE id = ?`
	selectSite  = `SELECT ` + siteColumns + ` FROM sites WHERE id = ?`
)

type SiteRepo struct {
	store *Store
}

func NewSiteRepo(store *Store) *SiteRepo {
	return &SiteRepo{store: store}
}

func siteNotFound(id string) *errors.Error {
	return errors.New(errors.NotFound, "site not found").WithDetail("siteId", id)
}

func siteConflict(id string) *errors.Error {
	return errors.New(errors.Conflict, "a site with this id already exists").WithDetail("siteId", id)
}

type siteColumnsJSON struct {
	capabilities string
	profiles     string
}

func encodeSiteColumns(s *site.Site) (siteColumnsJSON, error) {
	capabilities := s.Plugin.Capabilities
	if capabilities == nil {
		capabilities = []string{}
	}
	profiles := s.Defaults.ModelProfiles
	if profiles == nil {
		profiles = map[llm.Role]llm.ModelRef{}
	}

	encodedCapabilities, err := encodeJSON(capabilities)
	if err != nil {
		return siteColumnsJSON{}, err
	}
	encodedProfiles, err := encodeJSON(profiles)
	if err != nil {
		return siteColumnsJSON{}, err
	}
	return siteColumnsJSON{capabilities: encodedCapabilities, profiles: encodedProfiles}, nil
}

func (r *SiteRepo) Insert(ctx context.Context, s site.Site) error {
	encoded, err := encodeSiteColumns(&s)
	if err != nil {
		return err
	}
	_, err = execWrite(ctx, r.store.writeFrom(ctx), insertSite, []any{
		s.ID, s.Name, s.BaseURL, s.Username, s.SecretRef, string(s.Status), boolInt(s.AllowInsecure),
		boolInt(s.Plugin.Installed), s.Plugin.Version, encoded.capabilities, s.Plugin.SEOPlugin,
		nullString(s.Defaults.TemplateID), nullString(s.Defaults.LinkPolicyID), encoded.profiles,
		formatTime(s.CreatedAt), formatTime(s.UpdatedAt),
	}, siteConflict(s.ID), "insert the site")
	return err
}

func (r *SiteRepo) Update(ctx context.Context, s site.Site) error {
	encoded, err := encodeSiteColumns(&s)
	if err != nil {
		return err
	}
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), updateSite, []any{
		s.Name, s.BaseURL, s.Username, string(s.Status), boolInt(s.AllowInsecure),
		boolInt(s.Plugin.Installed), s.Plugin.Version, encoded.capabilities, s.Plugin.SEOPlugin,
		nullString(s.Defaults.TemplateID), nullString(s.Defaults.LinkPolicyID), encoded.profiles,
		formatTime(s.UpdatedAt), s.ID,
	}, nil, "update the site")
	return requireAffected(affected, err, siteNotFound(s.ID))
}

func (r *SiteRepo) Delete(ctx context.Context, id string) error {
	affected, err := execWrite(ctx, r.store.writeFrom(ctx), deleteSite, []any{id}, nil, "delete the site")
	return requireAffected(affected, err, siteNotFound(id))
}

func (r *SiteRepo) Get(ctx context.Context, id string) (site.Site, error) {
	return selectOne(ctx, r.store.execFrom(ctx), selectSite, []any{id}, scanSite, siteNotFound(id), "read the site")
}

func siteKeyset(q site.Query) paging.Keyset[site.Site] {
	key := paging.TimeKey[site.Site]("createdAt", "created_at", func(s site.Site) any { return s.CreatedAt })
	if q.Sort == site.SortName {
		key = paging.TextKey[site.Site]("name", "name", func(s site.Site) any { return s.Name })
	}
	return paging.Keyset[site.Site]{
		IDColumn: "id",
		ID:       func(s site.Site) string { return s.ID },
		Keys:     []paging.SortKey[site.Site]{key},
		Desc:     q.Desc,
	}
}

func (r *SiteRepo) List(ctx context.Context, q site.Query, page paging.Request) (paging.List[site.Site], error) {
	builder := squirrel.Select(siteColumns).From("sites")
	if q.Status != nil {
		builder = builder.Where(squirrel.Eq{"status": string(*q.Status)})
	}

	keyset := siteKeyset(q)
	keyed, err := keyset.Apply(builder, page)
	if err != nil {
		return paging.List[site.Site]{}, err
	}
	query, args, err := buildQuery(keyed, "sites")
	if err != nil {
		return paging.List[site.Site]{}, err
	}
	rows, err := selectAll(ctx, r.store.execFrom(ctx), query, args, scanSite, "list the sites")
	if err != nil {
		return paging.List[site.Site]{}, err
	}
	return keyset.Cut(rows, page)
}

func scanSite(rows *sql.Rows) (site.Site, error) {
	var (
		s                              site.Site
		status                         string
		allowInsecure, pluginInstalled int64
		capabilities, profiles         string
		templateID, policyID           sql.NullString
		createdAt, updatedAt           string
	)
	if err := rows.Scan(&s.ID, &s.Name, &s.BaseURL, &s.Username, &s.SecretRef, &status, &allowInsecure, &pluginInstalled,
		&s.Plugin.Version, &capabilities, &s.Plugin.SEOPlugin, &templateID, &policyID, &profiles, &createdAt, &updatedAt); err != nil {
		return site.Site{}, err
	}

	s.Status = site.Status(status)
	s.AllowInsecure = allowInsecure == 1
	s.Plugin.Installed = pluginInstalled == 1
	s.Defaults.TemplateID = optString(templateID)
	s.Defaults.LinkPolicyID = optString(policyID)
	if err := decodeJSON(capabilities, &s.Plugin.Capabilities, "decode the site plugin capabilities"); err != nil {
		return site.Site{}, err
	}
	if err := decodeJSON(profiles, &s.Defaults.ModelProfiles, "decode the site model profiles"); err != nil {
		return site.Site{}, err
	}

	var err error
	if s.CreatedAt, err = parseTime(createdAt); err != nil {
		return site.Site{}, err
	}
	if s.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return site.Site{}, err
	}
	return s, nil
}
