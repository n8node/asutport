package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n8node/asutport/internal/models"
)

type InstallationRepo struct {
	pool *pgxpool.Pool
}

func NewInstallationRepo(pool *pgxpool.Pool) *InstallationRepo {
	return &InstallationRepo{pool: pool}
}

type InstallationUpsertParams struct {
	ClientOrgID           uuid.UUID
	Name                  string
	SiteAddress           string
	Criticality           string
	SnapshotAllowed       bool
	EmergencyContactName  string
	EmergencyContactPhone string
	Environment           map[string]any
}

func (r *InstallationRepo) ListByClientOrg(ctx context.Context, orgID uuid.UUID) ([]models.Installation, error) {
	q := `SELECT id, client_org_id, name, site_address, criticality::text, snapshot_allowed,
			emergency_contact_name, emergency_contact_phone, environment, created_at, updated_at
		FROM installations
		WHERE client_org_id = $1
		ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list installations: %w", err)
	}
	defer rows.Close()
	var out []models.Installation
	for rows.Next() {
		item, err := scanInstallation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *InstallationRepo) GetByIDForOrg(ctx context.Context, id, orgID uuid.UUID) (*models.Installation, error) {
	q := `SELECT id, client_org_id, name, site_address, criticality::text, snapshot_allowed,
			emergency_contact_name, emergency_contact_phone, environment, created_at, updated_at
		FROM installations
		WHERE id = $1 AND client_org_id = $2`
	row := r.pool.QueryRow(ctx, q, id, orgID)
	item, err := scanInstallation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *InstallationRepo) Create(ctx context.Context, p InstallationUpsertParams) (*models.Installation, error) {
	env, err := json.Marshal(p.Environment)
	if err != nil {
		return nil, fmt.Errorf("marshal environment: %w", err)
	}
	id := uuid.New()
	criticality := normalizeCriticality(p.Criticality)
	q := `INSERT INTO installations (
			id, client_org_id, name, site_address, criticality, snapshot_allowed,
			emergency_contact_name, emergency_contact_phone, environment
		) VALUES ($1, $2, $3, $4, $5::production_criticality, $6, $7, $8, $9)
		RETURNING id, client_org_id, name, site_address, criticality::text, snapshot_allowed,
			emergency_contact_name, emergency_contact_phone, environment, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q,
		id, p.ClientOrgID, strings.TrimSpace(p.Name), strings.TrimSpace(p.SiteAddress),
		criticality, p.SnapshotAllowed, strings.TrimSpace(p.EmergencyContactName),
		strings.TrimSpace(p.EmergencyContactPhone), env,
	)
	return scanInstallation(row)
}

func (r *InstallationRepo) Update(ctx context.Context, id, orgID uuid.UUID, p InstallationUpsertParams) (*models.Installation, error) {
	env, err := json.Marshal(p.Environment)
	if err != nil {
		return nil, fmt.Errorf("marshal environment: %w", err)
	}
	criticality := normalizeCriticality(p.Criticality)
	q := `UPDATE installations SET
			name = $3,
			site_address = $4,
			criticality = $5::production_criticality,
			snapshot_allowed = $6,
			emergency_contact_name = $7,
			emergency_contact_phone = $8,
			environment = $9,
			updated_at = now()
		WHERE id = $1 AND client_org_id = $2
		RETURNING id, client_org_id, name, site_address, criticality::text, snapshot_allowed,
			emergency_contact_name, emergency_contact_phone, environment, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q,
		id, orgID, strings.TrimSpace(p.Name), strings.TrimSpace(p.SiteAddress),
		criticality, p.SnapshotAllowed, strings.TrimSpace(p.EmergencyContactName),
		strings.TrimSpace(p.EmergencyContactPhone), env,
	)
	item, err := scanInstallation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *InstallationRepo) CountByClientOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM installations WHERE client_org_id = $1`, orgID).Scan(&n)
	return n, err
}

type ProductUpsertParams struct {
	ManufacturerName string
	ProductName      string
	Kind             string
	Version          string
	Notes            string
}

func (r *InstallationRepo) ListProducts(ctx context.Context, installationID uuid.UUID) ([]models.InstallationProduct, error) {
	q := `SELECT id, installation_id, manufacturer_name, product_name, kind::text, version, notes, created_at, updated_at
		FROM installation_products
		WHERE installation_id = $1
		ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, q, installationID)
	if err != nil {
		return nil, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()
	var out []models.InstallationProduct
	for rows.Next() {
		item, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *InstallationRepo) GetProductForOrg(ctx context.Context, productID, orgID uuid.UUID) (*models.InstallationProduct, error) {
	q := `SELECT p.id, p.installation_id, p.manufacturer_name, p.product_name, p.kind::text, p.version, p.notes, p.created_at, p.updated_at
		FROM installation_products p
		JOIN installations i ON i.id = p.installation_id
		WHERE p.id = $1 AND i.client_org_id = $2`
	row := r.pool.QueryRow(ctx, q, productID, orgID)
	item, err := scanProduct(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *InstallationRepo) CreateProduct(ctx context.Context, installationID uuid.UUID, p ProductUpsertParams) (*models.InstallationProduct, error) {
	id := uuid.New()
	kind := normalizeProductKind(p.Kind)
	q := `INSERT INTO installation_products (
			id, installation_id, manufacturer_name, product_name, kind, version, notes
		) VALUES ($1, $2, $3, $4, $5::product_kind, $6, $7)
		RETURNING id, installation_id, manufacturer_name, product_name, kind::text, version, notes, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q,
		id, installationID,
		strings.TrimSpace(p.ManufacturerName), strings.TrimSpace(p.ProductName),
		kind, strings.TrimSpace(p.Version), strings.TrimSpace(p.Notes),
	)
	return scanProduct(row)
}

func (r *InstallationRepo) UpdateProduct(ctx context.Context, productID uuid.UUID, p ProductUpsertParams) (*models.InstallationProduct, error) {
	kind := normalizeProductKind(p.Kind)
	q := `UPDATE installation_products SET
			manufacturer_name = $2,
			product_name = $3,
			kind = $4::product_kind,
			version = $5,
			notes = $6,
			updated_at = now()
		WHERE id = $1
		RETURNING id, installation_id, manufacturer_name, product_name, kind::text, version, notes, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q,
		productID,
		strings.TrimSpace(p.ManufacturerName), strings.TrimSpace(p.ProductName),
		kind, strings.TrimSpace(p.Version), strings.TrimSpace(p.Notes),
	)
	item, err := scanProduct(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return item, err
}

func (r *InstallationRepo) DeleteProduct(ctx context.Context, productID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM installation_products WHERE id = $1`, productID)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *InstallationRepo) CountProductsByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM installation_products p JOIN installations i ON i.id = p.installation_id WHERE i.client_org_id = $1`,
		orgID,
	).Scan(&n)
	return n, err
}

type SupplyUpsertParams struct {
	SerialOrLicense string
	SupplierName    string
	IntegratorName  string
	PurchaseDate    *time.Time
	WarrantyUntil   *time.Time
	ContractRef     string
}

func (r *InstallationRepo) ListSupplyRecords(ctx context.Context, orgID uuid.UUID) ([]models.SupplyRecord, error) {
	q := `SELECT s.id, s.installation_product_id, s.serial_or_license, s.supplier_name, s.integrator_name,
			s.purchase_date, s.warranty_until, s.contract_ref, s.verify_status::text, s.created_at, s.updated_at
		FROM supply_records s
		JOIN installation_products p ON p.id = s.installation_product_id
		JOIN installations i ON i.id = p.installation_id
		WHERE i.client_org_id = $1
		ORDER BY s.created_at DESC`
	rows, err := r.pool.Query(ctx, q, orgID)
	if err != nil {
		return nil, fmt.Errorf("list supply records: %w", err)
	}
	defer rows.Close()
	var out []models.SupplyRecord
	for rows.Next() {
		item, err := scanSupply(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *InstallationRepo) ListSupplyByProduct(ctx context.Context, productID uuid.UUID) ([]models.SupplyRecord, error) {
	q := `SELECT id, installation_product_id, serial_or_license, supplier_name, integrator_name,
			purchase_date, warranty_until, contract_ref, verify_status::text, created_at, updated_at
		FROM supply_records
		WHERE installation_product_id = $1
		ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("list supply by product: %w", err)
	}
	defer rows.Close()
	var out []models.SupplyRecord
	for rows.Next() {
		item, err := scanSupply(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

func (r *InstallationRepo) CreateSupplyRecord(ctx context.Context, productID uuid.UUID, p SupplyUpsertParams) (*models.SupplyRecord, error) {
	id := uuid.New()
	q := `INSERT INTO supply_records (
			id, installation_product_id, serial_or_license, supplier_name, integrator_name,
			purchase_date, warranty_until, contract_ref
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, installation_product_id, serial_or_license, supplier_name, integrator_name,
			purchase_date, warranty_until, contract_ref, verify_status::text, created_at, updated_at`
	row := r.pool.QueryRow(ctx, q,
		id, productID,
		strings.TrimSpace(p.SerialOrLicense), strings.TrimSpace(p.SupplierName),
		strings.TrimSpace(p.IntegratorName), p.PurchaseDate, p.WarrantyUntil,
		strings.TrimSpace(p.ContractRef),
	)
	return scanSupply(row)
}

func (r *InstallationRepo) DeleteSupplyRecord(ctx context.Context, recordID, orgID uuid.UUID) error {
	q := `DELETE FROM supply_records s
		USING installation_products p, installations i
		WHERE s.id = $1 AND s.installation_product_id = p.id AND p.installation_id = i.id AND i.client_org_id = $2`
	tag, err := r.pool.Exec(ctx, q, recordID, orgID)
	if err != nil {
		return fmt.Errorf("delete supply record: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *InstallationRepo) CountSupplyByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM supply_records s
		 JOIN installation_products p ON p.id = s.installation_product_id
		 JOIN installations i ON i.id = p.installation_id
		 WHERE i.client_org_id = $1`,
		orgID,
	).Scan(&n)
	return n, err
}

func (r *InstallationRepo) DashboardSummary(ctx context.Context, orgID uuid.UUID) (*models.ClientDashboardSummary, error) {
	installations, err := r.CountByClientOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	products, err := r.CountProductsByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	supply, err := r.CountSupplyByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	profileComplete := false
	if installations > 0 {
		list, err := r.ListByClientOrg(ctx, orgID)
		if err != nil {
			return nil, err
		}
		for _, inst := range list {
			if strings.TrimSpace(inst.Name) != "" && strings.TrimSpace(inst.EmergencyContactPhone) != "" {
				profileComplete = true
				break
			}
		}
	}
	coverage := 0
	if profileComplete {
		coverage += 40
	}
	if products > 0 {
		coverage += 30
	}
	if supply > 0 {
		coverage += 30
	}
	if coverage > 100 {
		coverage = 100
	}
	return &models.ClientDashboardSummary{
		InstallationsCount: installations,
		ProductsCount:      products,
		SupplyRecordsCount: supply,
		ProfileComplete:    profileComplete,
		CoveragePercent:    coverage,
	}, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanInstallation(row scannable) (*models.Installation, error) {
	var item models.Installation
	err := row.Scan(
		&item.ID, &item.ClientOrgID, &item.Name, &item.SiteAddress, &item.Criticality,
		&item.SnapshotAllowed, &item.EmergencyContactName, &item.EmergencyContactPhone,
		&item.Environment, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func scanProduct(row scannable) (*models.InstallationProduct, error) {
	var item models.InstallationProduct
	err := row.Scan(
		&item.ID, &item.InstallationID, &item.ManufacturerName, &item.ProductName,
		&item.Kind, &item.Version, &item.Notes, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func scanSupply(row scannable) (*models.SupplyRecord, error) {
	var item models.SupplyRecord
	err := row.Scan(
		&item.ID, &item.InstallationProductID, &item.SerialOrLicense, &item.SupplierName,
		&item.IntegratorName, &item.PurchaseDate, &item.WarrantyUntil, &item.ContractRef,
		&item.VerifyStatus, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func normalizeCriticality(raw string) string {
	switch strings.TrimSpace(raw) {
	case "continuous", "batch", "auxiliary":
		return strings.TrimSpace(raw)
	default:
		return "batch"
	}
}

func normalizeProductKind(raw string) string {
	switch strings.TrimSpace(raw) {
	case "plc", "scada", "hmi", "drive", "sensor", "network", "other":
		return strings.TrimSpace(raw)
	default:
		return "other"
	}
}
