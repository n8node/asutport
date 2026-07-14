package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/models"
	"github.com/n8node/asutport/internal/repository"
	"github.com/n8node/asutport/internal/service"
)

type ClientHandler struct {
	installations *repository.InstallationRepo
	tickets       *service.TicketService
	orgs          *repository.OrgRepo
}

func NewClientHandler(
	installations *repository.InstallationRepo,
	tickets *service.TicketService,
	orgs *repository.OrgRepo,
) *ClientHandler {
	return &ClientHandler{installations: installations, tickets: tickets, orgs: orgs}
}

func (h *ClientHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	p, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	summary, err := h.installations.DashboardSummary(r.Context(), org.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load dashboard")
		return
	}
	openTickets, err := h.tickets.CountOpenByClientOrg(r.Context(), org.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load dashboard")
		return
	}
	slaActive, err := h.tickets.CountSLAActiveByClientOrg(r.Context(), org.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load dashboard")
		return
	}
	summary.OpenTicketsCount = openTickets
	summary.SLAActiveCount = slaActive
	_ = p
	WriteJSON(w, http.StatusOK, map[string]any{"data": dashboardSummaryDTO(summary)})
}

func (h *ClientHandler) ListInstallations(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	list, err := h.installations.ListByClientOrg(r.Context(), org.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list installations")
		return
	}
	items := make([]map[string]any, 0, len(list))
	for _, item := range list {
		items = append(items, installationDTO(&item))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

type installationReq struct {
	Name                  string         `json:"name"`
	SiteAddress           string         `json:"site_address"`
	Criticality           string         `json:"criticality"`
	SnapshotAllowed       bool           `json:"snapshot_allowed"`
	EmergencyContactName  string         `json:"emergency_contact_name"`
	EmergencyContactPhone string         `json:"emergency_contact_phone"`
	Environment           map[string]any `json:"environment"`
}

func (h *ClientHandler) CreateInstallation(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	var req installationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "укажите название площадки")
		return
	}
	item, err := h.installations.Create(r.Context(), repository.InstallationUpsertParams{
		ClientOrgID:           org.ID,
		Name:                  req.Name,
		SiteAddress:           req.SiteAddress,
		Criticality:           req.Criticality,
		SnapshotAllowed:       req.SnapshotAllowed,
		EmergencyContactName:  req.EmergencyContactName,
		EmergencyContactPhone: req.EmergencyContactPhone,
		Environment:           req.Environment,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not create installation")
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"data": installationDTO(item)})
}

func (h *ClientHandler) UpdateInstallation(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "installationID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid installation id")
		return
	}
	var req installationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "укажите название площадки")
		return
	}
	item, err := h.installations.Update(r.Context(), id, org.ID, repository.InstallationUpsertParams{
		ClientOrgID:           org.ID,
		Name:                  req.Name,
		SiteAddress:           req.SiteAddress,
		Criticality:           req.Criticality,
		SnapshotAllowed:       req.SnapshotAllowed,
		EmergencyContactName:  req.EmergencyContactName,
		EmergencyContactPhone: req.EmergencyContactPhone,
		Environment:           req.Environment,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "установка не найдена")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not update installation")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": installationDTO(item)})
}

func (h *ClientHandler) ListProducts(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	installationID, err := uuid.Parse(chi.URLParam(r, "installationID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid installation id")
		return
	}
	if _, err := h.installations.GetByIDForOrg(r.Context(), installationID, org.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "установка не найдена")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load installation")
		return
	}
	list, err := h.installations.ListProducts(r.Context(), installationID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list products")
		return
	}
	items := make([]map[string]any, 0, len(list))
	for _, item := range list {
		items = append(items, productDTO(&item))
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

type productReq struct {
	ManufacturerName string `json:"manufacturer_name"`
	ProductName      string `json:"product_name"`
	Kind             string `json:"kind"`
	Version          string `json:"version"`
	Notes            string `json:"notes"`
}

func (h *ClientHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	installationID, err := uuid.Parse(chi.URLParam(r, "installationID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid installation id")
		return
	}
	if _, err := h.installations.GetByIDForOrg(r.Context(), installationID, org.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "установка не найдена")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load installation")
		return
	}
	var req productReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	if strings.TrimSpace(req.ProductName) == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "укажите название продукта")
		return
	}
	item, err := h.installations.CreateProduct(r.Context(), installationID, repository.ProductUpsertParams{
		ManufacturerName: req.ManufacturerName,
		ProductName:      req.ProductName,
		Kind:             req.Kind,
		Version:          req.Version,
		Notes:            req.Notes,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not create product")
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"data": productDTO(item)})
}

func (h *ClientHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	productID, err := uuid.Parse(chi.URLParam(r, "productID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid product id")
		return
	}
	if _, err := h.installations.GetProductForOrg(r.Context(), productID, org.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "продукт не найден")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load product")
		return
	}
	var req productReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	if strings.TrimSpace(req.ProductName) == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "укажите название продукта")
		return
	}
	item, err := h.installations.UpdateProduct(r.Context(), productID, repository.ProductUpsertParams{
		ManufacturerName: req.ManufacturerName,
		ProductName:      req.ProductName,
		Kind:             req.Kind,
		Version:          req.Version,
		Notes:            req.Notes,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not update product")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": productDTO(item)})
}

func (h *ClientHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	productID, err := uuid.Parse(chi.URLParam(r, "productID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid product id")
		return
	}
	if _, err := h.installations.GetProductForOrg(r.Context(), productID, org.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "продукт не найден")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load product")
		return
	}
	if err := h.installations.DeleteProduct(r.Context(), productID); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not delete product")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "ok"}})
}

func (h *ClientHandler) ListSupplyRecords(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	list, err := h.installations.ListSupplyRecords(r.Context(), org.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list supply records")
		return
	}
	products, _ := h.installations.ListByClientOrg(r.Context(), org.ID)
	productNames := map[string]string{}
	for _, inst := range products {
		prods, _ := h.installations.ListProducts(r.Context(), inst.ID)
		for _, p := range prods {
			productNames[p.ID.String()] = p.ProductName
		}
	}
	items := make([]map[string]any, 0, len(list))
	for _, item := range list {
		dto := supplyRecordDTO(&item)
		dto["product_name"] = productNames[item.InstallationProductID.String()]
		items = append(items, dto)
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": items})
}

type supplyReq struct {
	InstallationProductID string `json:"installation_product_id"`
	SerialOrLicense       string `json:"serial_or_license"`
	SupplierName          string `json:"supplier_name"`
	IntegratorName        string `json:"integrator_name"`
	PurchaseDate          string `json:"purchase_date"`
	WarrantyUntil         string `json:"warranty_until"`
	ContractRef           string `json:"contract_ref"`
}

func (h *ClientHandler) CreateSupplyRecord(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	var req supplyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	productID, err := uuid.Parse(strings.TrimSpace(req.InstallationProductID))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "укажите продукт")
		return
	}
	if strings.TrimSpace(req.SerialOrLicense) == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "укажите серийный номер или лицензию")
		return
	}
	if _, err := h.installations.GetProductForOrg(r.Context(), productID, org.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "продукт не найден")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load product")
		return
	}
	item, err := h.installations.CreateSupplyRecord(r.Context(), productID, repository.SupplyUpsertParams{
		SerialOrLicense: req.SerialOrLicense,
		SupplierName:  req.SupplierName,
		IntegratorName: req.IntegratorName,
		PurchaseDate:  parseOptionalDate(req.PurchaseDate),
		WarrantyUntil: parseOptionalDate(req.WarrantyUntil),
		ContractRef:   req.ContractRef,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not create supply record")
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"data": supplyRecordDTO(item)})
}

func (h *ClientHandler) DeleteSupplyRecord(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	recordID, err := uuid.Parse(chi.URLParam(r, "recordID"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid record id")
		return
	}
	if err := h.installations.DeleteSupplyRecord(r.Context(), recordID, org.ID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			WriteError(w, http.StatusNotFound, "NOT_FOUND", "запись не найдена")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not delete record")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]string{"status": "ok"}})
}

func (h *ClientHandler) ListTickets(w http.ResponseWriter, r *http.Request) {
	_, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	list, total, err := h.tickets.ListByClientOrg(r.Context(), org.ID, limit, offset)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not list tickets")
		return
	}
	items := make([]map[string]any, 0, len(list))
	for _, t := range list {
		items = append(items, clientTicketDTO(&t))
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{"total": total},
	})
}

type createTicketReq struct {
	Subject        string `json:"subject"`
	Type           string `json:"type"`
	Priority       string `json:"priority"`
	InstallationID string `json:"installation_id"`
	Text           string `json:"text"`
}

func (h *ClientHandler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	p, org, ok := h.requireActiveClient(w, r)
	if !ok {
		return
	}
	var req createTicketReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "invalid json body")
		return
	}
	req.Subject = strings.TrimSpace(req.Subject)
	req.Text = strings.TrimSpace(req.Text)
	if req.Subject == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "укажите тему обращения")
		return
	}
	var installationID *uuid.UUID
	if strings.TrimSpace(req.InstallationID) != "" {
		id, err := uuid.Parse(req.InstallationID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", "некорректная установка")
			return
		}
		if _, err := h.installations.GetByIDForOrg(r.Context(), id, org.ID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				WriteError(w, http.StatusNotFound, "NOT_FOUND", "установка не найдена")
				return
			}
			WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load installation")
			return
		}
		installationID = &id
	}
	ticket, err := h.tickets.CreateSupportTicket(r.Context(), service.CreateSupportTicketInput{
		ClientOrgID:     org.ID,
		InstallationID:  installationID,
		Subject:         req.Subject,
		Type:            req.Type,
		Priority:        req.Priority,
		CreatedByUserID: p.UserID,
		InitialText:     req.Text,
	})
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_ERROR", userFacingErr(err))
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"data": clientTicketDTO(ticket)})
}

func (h *ClientHandler) requireActiveClient(w http.ResponseWriter, r *http.Request) (*auth.Principal, *models.Organization, bool) {
	p, ok := auth.PrincipalFromContext(r.Context())
	if !ok {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing authentication")
		return nil, nil, false
	}
	org, err := h.orgs.GetByID(r.Context(), p.OrgID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "could not load organization")
		return nil, nil, false
	}
	if org.Type != "client_org" {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "доступно только для кабинета эксплуатации")
		return nil, nil, false
	}
	if org.ReviewStatus != "active" {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "организация ещё не активирована")
		return nil, nil, false
	}
	return p, org, true
}

func dashboardSummaryDTO(s *models.ClientDashboardSummary) map[string]any {
	if s == nil {
		return nil
	}
	return map[string]any{
		"installations_count":  s.InstallationsCount,
		"open_tickets_count":   s.OpenTicketsCount,
		"sla_active_count":     s.SLAActiveCount,
		"coverage_percent":     s.CoveragePercent,
		"profile_complete":     s.ProfileComplete,
		"products_count":       s.ProductsCount,
		"supply_records_count": s.SupplyRecordsCount,
	}
}

func installationDTO(item *models.Installation) map[string]any {
	if item == nil {
		return nil
	}
	var env map[string]any
	_ = json.Unmarshal(item.Environment, &env)
	return map[string]any{
		"id":                      item.ID.String(),
		"name":                    item.Name,
		"site_address":            item.SiteAddress,
		"criticality":             item.Criticality,
		"snapshot_allowed":        item.SnapshotAllowed,
		"emergency_contact_name":  item.EmergencyContactName,
		"emergency_contact_phone": item.EmergencyContactPhone,
		"environment":             env,
		"created_at":              item.CreatedAt,
		"updated_at":              item.UpdatedAt,
	}
}

func productDTO(item *models.InstallationProduct) map[string]any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"id":                item.ID.String(),
		"installation_id":   item.InstallationID.String(),
		"manufacturer_name": item.ManufacturerName,
		"product_name":      item.ProductName,
		"kind":              item.Kind,
		"version":           item.Version,
		"notes":             item.Notes,
		"created_at":        item.CreatedAt,
		"updated_at":        item.UpdatedAt,
	}
}

func supplyRecordDTO(item *models.SupplyRecord) map[string]any {
	if item == nil {
		return nil
	}
	dto := map[string]any{
		"id":                      item.ID.String(),
		"installation_product_id": item.InstallationProductID.String(),
		"serial_or_license":       item.SerialOrLicense,
		"supplier_name":           item.SupplierName,
		"integrator_name":         item.IntegratorName,
		"contract_ref":            item.ContractRef,
		"verify_status":           item.VerifyStatus,
		"created_at":              item.CreatedAt,
		"updated_at":              item.UpdatedAt,
	}
	if item.PurchaseDate != nil {
		dto["purchase_date"] = item.PurchaseDate.Format("2006-01-02")
	}
	if item.WarrantyUntil != nil {
		dto["warranty_until"] = item.WarrantyUntil.Format("2006-01-02")
	}
	return dto
}

func clientTicketDTO(t *models.Ticket) map[string]any {
	if t == nil {
		return nil
	}
	dto := map[string]any{
		"id":         t.ID.String(),
		"type":       t.Type,
		"priority":   t.Priority,
		"status":     t.Status,
		"subject":    t.Subject,
		"created_at": t.CreatedAt,
		"updated_at": t.UpdatedAt,
	}
	if t.InstallationID != nil {
		dto["installation_id"] = t.InstallationID.String()
	}
	if t.BallOwnerOrgID != nil {
		dto["ball_owner_org_id"] = t.BallOwnerOrgID.String()
		dto["ball_owner_org_name"] = t.BallOwnerOrgName
	}
	if t.AssignedTargetOrgID != nil {
		dto["assigned_target_org_id"] = t.AssignedTargetOrgID.String()
		dto["assigned_target_org_name"] = t.AssignedTargetName
	}
	if t.SLAReactionDeadline != nil {
		dto["sla_reaction_deadline"] = t.SLAReactionDeadline
	}
	return dto
}

func parseOptionalDate(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return nil
	}
	return &t
}
