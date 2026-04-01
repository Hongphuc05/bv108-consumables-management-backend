package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"bv108-consumables-management-backend/internal/models"
	"bv108-consumables-management-backend/internal/realtime"
	"bv108-consumables-management-backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type OrderHandler struct {
	repo               *models.OrderRepository
	invoiceMatchRepo   *models.InvoiceReconciliationRepository
	unreadRepo         *models.OrderUnreadRepository
	companyContactRepo *models.CompanyContactRepository
	userRepo           *models.UserRepository
	jwtSecret          []byte
	mailer             services.OrderEmailSender
	hub                *realtime.Hub
}

type CreateForecastOrdersRequest struct {
	Items []CreateOrderItemRequest `json:"items" binding:"required"`
}

type CreateOrderItemRequest struct {
	NhaThau    string `json:"nhaThau"`
	MaQuanLy   string `json:"maQuanLy"`
	MaVtytCu   string `json:"maVtytCu"`
	TenVtytBv  string `json:"tenVtytBv"`
	MaHieu     string `json:"maHieu"`
	HangSx     string `json:"hangSx"`
	DonViTinh  string `json:"donViTinh"`
	QuyCach    string `json:"quyCach"`
	DotGoiHang int    `json:"dotGoiHang"`
	Email      string `json:"email"`
}

type PlaceOrdersRequest struct {
	OrderIDs []int64 `json:"orderIds" binding:"required"`
}

type MarkGroupsSeenRequest struct {
	GroupKeys []string `json:"groupKeys" binding:"required"`
}

type SaveInvoiceReconciliationsBulkRequest struct {
	Items []SaveInvoiceReconciliationItemRequest `json:"items" binding:"required"`
}

type SaveInvoiceReconciliationItemRequest struct {
	ID     int64  `json:"id"`
	Action string `json:"action"`
	Note   string `json:"note"`
	Status string `json:"status"`
}

func NewOrderHandler(repo *models.OrderRepository, invoiceMatchRepo *models.InvoiceReconciliationRepository, unreadRepo *models.OrderUnreadRepository, companyContactRepo *models.CompanyContactRepository, userRepo *models.UserRepository, jwtSecret string, mailer services.OrderEmailSender, hub *realtime.Hub) *OrderHandler {
	return &OrderHandler{
		repo:               repo,
		invoiceMatchRepo:   invoiceMatchRepo,
		unreadRepo:         unreadRepo,
		companyContactRepo: companyContactRepo,
		userRepo:           userRepo,
		jwtSecret:          []byte(jwtSecret),
		mailer:             mailer,
		hub:                hub,
	}
}

func (h *OrderHandler) SaveInvoiceReconciliations(c *gin.Context) {
	if h.invoiceMatchRepo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "UNAVAILABLE", Message: "Invoice reconciliation repository is not configured"})
		return
	}

	if _, err := h.getCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req SaveInvoiceReconciliationsBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid invoice reconciliation payload"})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "At least one reconciliation item is required"})
		return
	}

	noteUpdates := make([]models.UpdateInvoiceReconciliationNoteInput, 0, len(req.Items))
	statusUpdates := make([]models.UpdateInvoiceReconciliationStatusInput, 0, len(req.Items))
	for _, item := range req.Items {
		if item.ID <= 0 {
			continue
		}

		switch strings.ToLower(strings.TrimSpace(item.Action)) {
		case "note":
			noteUpdates = append(noteUpdates, models.UpdateInvoiceReconciliationNoteInput{
				ID:   item.ID,
				Note: strings.TrimSpace(item.Note),
			})
		case "status":
			normalizedStatus := strings.ToLower(strings.TrimSpace(item.Status))
			switch normalizedStatus {
			case "", "done", "xong":
				statusUpdates = append(statusUpdates, models.UpdateInvoiceReconciliationStatusInput{
					ID:     item.ID,
					Status: models.InvoiceReconciliationStatusDone,
				})
			case "waiting", "ch\u1edd":
				statusUpdates = append(statusUpdates, models.UpdateInvoiceReconciliationStatusInput{
					ID:     item.ID,
					Status: models.InvoiceReconciliationStatusPending,
				})
			}
		}
	}

	if len(noteUpdates) == 0 && len(statusUpdates) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "No valid invoice reconciliation items to save", "count": 0})
		return
	}

	var updatedCount int64
	if len(noteUpdates) > 0 {
		count, err := h.invoiceMatchRepo.UpdateNotesBulk(noteUpdates)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
			return
		}
		updatedCount += count
	}

	if len(statusUpdates) > 0 {
		count, err := h.invoiceMatchRepo.UpdateStatusesBulk(statusUpdates)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
			return
		}
		updatedCount += count
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invoice reconciliation updated", "count": updatedCount})
}

func (h *OrderHandler) GetInvoiceReconciliationHistory(c *gin.Context) {
	if h.invoiceMatchRepo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "UNAVAILABLE", Message: "Invoice reconciliation repository is not configured"})
		return
	}

	if _, err := h.getCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	month, err := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(time.Now().Month()))))
	if err != nil || month < 1 || month > 12 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "month must be from 1 to 12"})
		return
	}

	year, err := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(time.Now().Year())))
	if err != nil || year < 2000 || year > 3000 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "year is invalid"})
		return
	}

	records, err := h.invoiceMatchRepo.ListByMonthYear(month, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": records, "month": month, "year": year})
}

func (h *OrderHandler) GetMatchedInvoiceNumbers(c *gin.Context) {
	if h.invoiceMatchRepo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "UNAVAILABLE", Message: "Invoice reconciliation repository is not configured"})
		return
	}

	if _, err := h.getCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	allParam := strings.TrimSpace(c.Query("all"))
	if allParam != "" {
		if parsed, err := strconv.ParseBool(allParam); err == nil && parsed {
			invoiceNumbers, err := h.invoiceMatchRepo.ListAllMatchedInvoiceNumbers()
			if err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"data": invoiceNumbers, "all": true})
			return
		}
	}

	month, err := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(time.Now().Month()))))
	if err != nil || month < 1 || month > 12 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "month must be from 1 to 12"})
		return
	}

	year, err := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(time.Now().Year())))
	if err != nil || year < 2000 || year > 3000 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "year is invalid"})
		return
	}

	invoiceNumbers, err := h.invoiceMatchRepo.ListMatchedInvoiceNumbers(month, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": invoiceNumbers, "month": month, "year": year})
}

func (h *OrderHandler) GetMatchedOrderReconciliations(c *gin.Context) {
	if h.invoiceMatchRepo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "UNAVAILABLE", Message: "Invoice reconciliation repository is not configured"})
		return
	}

	if _, err := h.getCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	records, err := h.invoiceMatchRepo.ListAllReconciliations()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": records})
}

func (h *OrderHandler) SearchCompanyContacts(c *gin.Context) {
	if h.companyContactRepo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "UNAVAILABLE", Message: "Company contact repository is not configured"})
		return
	}

	if _, err := h.getCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "keyword is required"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "8"))
	if limit < 1 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}

	contacts, err := h.companyContactRepo.Search(keyword, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": contacts})
}

func (h *OrderHandler) GetPendingOrders(c *gin.Context) {
	if _, err := h.getCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	orders, err := h.repo.ListPendingOrders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": orders})
}

func (h *OrderHandler) GetOrderHistory(c *gin.Context) {
	if _, err := h.getCurrentUser(c); err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	history, err := h.repo.ListOrderHistory()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": history})
}

func (h *OrderHandler) CreateForecastOrders(c *gin.Context) {
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req CreateForecastOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid forecast order payload"})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "At least one order item is required"})
		return
	}

	inputs := make([]models.CreatePendingOrderInput, 0, len(req.Items))
	approvalTime := time.Now().Format(time.RFC3339)
	createdGroupKeys := make([]string, 0, len(req.Items))
	for _, item := range req.Items {
		if item.DotGoiHang < 1 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "dotGoiHang must be greater than 0"})
			return
		}

		groupKey := buildPendingOrderGroupKey(item.NhaThau, approvalTime)

		inputs = append(inputs, models.CreatePendingOrderInput{
			NhaThau:      sanitizeText(item.NhaThau),
			MaQuanLy:     sanitizeText(item.MaQuanLy),
			MaVtytCu:     sanitizeText(item.MaVtytCu),
			TenVtytBv:    sanitizeText(item.TenVtytBv),
			MaHieu:       sanitizeText(item.MaHieu),
			HangSx:       sanitizeText(item.HangSx),
			DonViTinh:    sanitizeText(item.DonViTinh),
			QuyCach:      sanitizeText(item.QuyCach),
			DotGoiHang:   item.DotGoiHang,
			Email:        sanitizeText(item.Email),
			Source:       models.OrderSourceForecast,
			GroupKey:     groupKey,
			Approver:     &models.OrderActor{ID: currentUser.ID, Username: currentUser.Username, Email: currentUser.Email},
			CreatedBy:    models.OrderActor{ID: currentUser.ID, Username: currentUser.Username, Email: currentUser.Email},
			ApprovalTime: approvalTime,
		})
		createdGroupKeys = append(createdGroupKeys, groupKey)
	}

	if err := h.repo.AddForecastOrders(inputs); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	if h.hub != nil {
		h.hub.Broadcast("orders.new_pending", gin.H{
			"groupKeys": uniqueNonEmptyStrings(createdGroupKeys),
			"createdAt": approvalTime,
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Forecast orders added to pending list successfully",
		"count":   len(inputs),
	})
}

func (h *OrderHandler) CreateManualOrder(c *gin.Context) {
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req CreateOrderItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid manual order payload"})
		return
	}

	if req.DotGoiHang < 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "dotGoiHang must be greater than 0"})
		return
	}

	if sanitizeText(req.NhaThau) == "" || sanitizeText(req.MaVtytCu) == "" || sanitizeText(req.TenVtytBv) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "nhaThau, maVtytCu, tenVtytBv are required"})
		return
	}

	input := models.CreatePendingOrderInput{
		NhaThau:    sanitizeText(req.NhaThau),
		MaQuanLy:   sanitizeText(req.MaQuanLy),
		MaVtytCu:   sanitizeText(req.MaVtytCu),
		TenVtytBv:  sanitizeText(req.TenVtytBv),
		MaHieu:     sanitizeText(req.MaHieu),
		HangSx:     sanitizeText(req.HangSx),
		DonViTinh:  sanitizeText(req.DonViTinh),
		QuyCach:    sanitizeText(req.QuyCach),
		DotGoiHang: req.DotGoiHang,
		Email:      sanitizeText(req.Email),
		Source:     models.OrderSourceManual,
		CreatedBy:  models.OrderActor{ID: currentUser.ID, Username: currentUser.Username, Email: currentUser.Email},
	}

	if err := h.repo.AddManualOrder(input); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Manual order added to pending list successfully"})
}

func (h *OrderHandler) PlaceOrders(c *gin.Context) {
	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req PlaceOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid place order payload"})
		return
	}

	if len(req.OrderIDs) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "At least one order id is required"})
		return
	}

	pendingOrders, err := h.repo.GetPendingOrdersByIDs(req.OrderIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	if len(pendingOrders) == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "NOT_FOUND", Message: "No pending orders found"})
		return
	}

	if countUniqueOrderIDs(req.OrderIDs) != len(pendingOrders) {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "NOT_FOUND", Message: "Some pending orders were not found"})
		return
	}

	if err := h.sendPlacedOrderEmails(pendingOrders); err != nil {
		c.JSON(http.StatusBadGateway, ErrorResponse{Error: "EMAIL_SEND_ERROR", Message: err.Error()})
		return
	}

	placedCount, err := h.repo.PlaceOrders(req.OrderIDs, models.OrderActor{
		ID:       currentUser.ID,
		Username: currentUser.Username,
		Email:    currentUser.Email,
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		if err.Error() == "no pending orders found" {
			statusCode = http.StatusNotFound
		}

		c.JSON(statusCode, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Orders placed and email sent successfully",
		"placedCount": placedCount,
	})
}

func (h *OrderHandler) GetUnreadSnapshot(c *gin.Context) {
	if h.unreadRepo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "UNAVAILABLE", Message: "Unread repository is not configured"})
		return
	}

	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	snapshot, err := h.unreadRepo.GetUnreadSnapshot(currentUser.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": snapshot})
}

func (h *OrderHandler) MarkSupplierAlertSeen(c *gin.Context) {
	if h.unreadRepo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "UNAVAILABLE", Message: "Unread repository is not configured"})
		return
	}

	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	now := time.Now().UTC()
	if err := h.unreadRepo.MarkSupplierAlertSeen(currentUser.ID, now); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	h.pushUnreadSnapshot(currentUser.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Supplier alert marked as seen"})
}

func (h *OrderHandler) MarkGroupsSeen(c *gin.Context) {
	if h.unreadRepo == nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{Error: "UNAVAILABLE", Message: "Unread repository is not configured"})
		return
	}

	currentUser, err := h.getCurrentUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "UNAUTHORIZED", Message: err.Error()})
		return
	}

	var req MarkGroupsSeenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "Invalid group seen payload"})
		return
	}

	if len(req.GroupKeys) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "groupKeys is required"})
		return
	}

	groupKeys := uniqueNonEmptyStrings(req.GroupKeys)
	if len(groupKeys) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "INVALID_REQUEST", Message: "groupKeys is required"})
		return
	}

	now := time.Now().UTC()
	if err := h.unreadRepo.MarkGroupsSeen(currentUser.ID, groupKeys, now); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "DATABASE_ERROR", Message: err.Error()})
		return
	}

	h.pushUnreadSnapshot(currentUser.ID)
	c.JSON(http.StatusOK, gin.H{"message": "Groups marked as seen", "count": len(groupKeys)})
}

func (h *OrderHandler) sendPlacedOrderEmails(orders []models.PendingOrder) error {
	if h.mailer == nil {
		return fmt.Errorf("email sender is not configured")
	}

	type emailGroup struct {
		supplierName string
		email        string
		items        []services.OrderEmailItem
	}

	groups := make(map[string]emailGroup)
	for _, order := range orders {
		email := strings.TrimSpace(order.Email)
		supplierName := strings.TrimSpace(order.NhaThau)
		if email == "" {
			if supplierName == "" {
				return fmt.Errorf("missing company email")
			}
			return fmt.Errorf("missing company email for %s", supplierName)
		}

		key := strings.ToLower(email) + "|" + strings.ToLower(supplierName)
		group, exists := groups[key]
		if !exists {
			group = emailGroup{
				supplierName: supplierName,
				email:        email,
				items:        make([]services.OrderEmailItem, 0, 4),
			}
		}

		group.items = append(group.items, services.OrderEmailItem{
			Index:     len(group.items) + 1,
			TenVatTu:  strings.TrimSpace(order.TenVtytBv),
			MaVatTu:   strings.TrimSpace(order.MaVtytCu),
			DonViTinh: strings.TrimSpace(order.DonViTinh),
			SoLuong:   order.DotGoiHang,
		})
		groups[key] = group
	}

	for _, group := range groups {
		if err := h.mailer.SendPlacedOrderEmail(group.email, group.supplierName, group.items); err != nil {
			return err
		}
	}

	return nil
}

func (h *OrderHandler) getCurrentUser(c *gin.Context) (*models.UserProfile, error) {
	userID, err := h.getUserIDFromAuthorizationHeader(c)
	if err != nil {
		return nil, err
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}

	profile := user.ToProfile()
	return &profile, nil
}

func (h *OrderHandler) getUserIDFromAuthorizationHeader(c *gin.Context) (int64, error) {
	authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
	if authHeader == "" {
		return 0, fmt.Errorf("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return 0, fmt.Errorf("invalid authorization header format")
	}

	tokenString := strings.TrimSpace(parts[1])
	if tokenString == "" {
		return 0, fmt.Errorf("missing bearer token")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return h.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return 0, fmt.Errorf("invalid or expired token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims")
	}

	subValue, exists := claims["sub"]
	if !exists {
		return 0, fmt.Errorf("missing subject in token")
	}

	userID, err := convertClaimToInt64(subValue)
	if err != nil {
		return 0, fmt.Errorf("invalid subject in token")
	}

	return userID, nil
}

func sanitizeText(value string) string {
	return strings.TrimSpace(value)
}

func countUniqueOrderIDs(orderIDs []int64) int {
	set := make(map[int64]struct{}, len(orderIDs))
	for _, orderID := range orderIDs {
		set[orderID] = struct{}{}
	}
	return len(set)
}

func buildPendingOrderGroupKey(nhaThau, approvalTime string) string {
	normalizedCompany := strings.ToLower(strings.TrimSpace(nhaThau))
	if normalizedCompany == "" {
		normalizedCompany = "unknown"
	}
	return fmt.Sprintf("%s__%s", normalizedCompany, strings.TrimSpace(approvalTime))
}

func uniqueNonEmptyStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}

	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func parseOptionalRFC3339(value string) (*time.Time, bool) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, false
	}

	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, false
	}

	return &parsed, true
}

func (h *OrderHandler) pushUnreadSnapshot(userID int64) {
	if h.unreadRepo == nil || h.hub == nil {
		return
	}

	snapshot, err := h.unreadRepo.GetUnreadSnapshot(userID)
	if err != nil {
		return
	}

	h.hub.SendToUser(userID, "orders.unread_updated", snapshot)
}
