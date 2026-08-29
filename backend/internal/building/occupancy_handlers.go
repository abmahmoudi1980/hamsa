package building

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"hamsa/internal/audit"
	"hamsa/internal/platform/httpx"
)

// Persian handler messages for US3 internal failures.
const (
	msgPersonListFailed = "خطا در دریافت فهرست افراد."
	msgOccListFailed    = "خطا در دریافت سابقه سکونت."
	msgCountListFailed  = "خطا در دریافت سابقه تعداد ساکن."
)

// RegisterOccupancy mounts US3 routes under r (expected group: /api/v1 with
// authMW). Resident changes are audited as `resident.changed` with the unit
// as the audited object (FR-038); person CRUD is audited per person.
func RegisterOccupancy(r *gin.RouterGroup, svc *Service, aud *audit.Service) {
	buildings := r.Group("/buildings")
	{
		idB := buildings.Group("/:id")
		{
			idB.GET("/persons", listPersons(svc))
			idB.POST("/persons", aud.Middleware("person.create", "person"), createPerson(svc))
		}
	}

	persons := r.Group("/persons")
	{
		persons.PUT("/:id", aud.Middleware("person.update", "person"), updatePerson(svc))
		persons.DELETE("/:id", aud.Middleware("person.delete", "person"), deletePerson(svc))
	}

	units := r.Group("/units")
	{
		units.GET("/:id/occupancies", listOccupancies(svc))
		units.POST("/:id/occupancies", aud.Middleware("resident.changed", "occupancy"), addOccupancy(svc))
		units.GET("/:id/occupant-count", listOccupantCounts(svc))
		units.POST("/:id/occupant-count", recordOccupantCount(svc))
	}

	r.PATCH("/occupancies/:id", aud.Middleware("resident.changed", "occupancy"), endOccupancy(svc))
}

// setOccupancyAuditCtx publishes the unit as the audited object plus the
// after-snapshot for the audit middleware.
func setOccupancyAuditCtx(c *gin.Context, unitID uuid.UUID, after any) {
	c.Set(audit.CtxObjectID, unitID)
	if after != nil {
		c.Set(audit.CtxAfter, after)
	}
}

// --- persons -------------------------------------------------------------------

func listPersons(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c)
		if !ok {
			return
		}
		page := atoiDefault(c.Query("page"), 1)
		size := atoiDefault(c.Query("page_size"), 20)
		items, total, err := svc.ListPersons(c.Request.Context(), mgr, buildingID, c.Query("q"), page, size)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"items": items, "page": page, "page_size": size, "total": total,
		})
	}
}

func createPerson(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		buildingID, ok := parseID(c)
		if !ok {
			return
		}
		var in PersonInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest(msgPersonNameRequired))
			return
		}
		p, err := svc.CreatePerson(c.Request.Context(), mgr, buildingID, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, p.ID)
		c.Set(audit.CtxAfter, p)
		c.JSON(http.StatusCreated, p)
	}
}

func updatePerson(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		var in PersonInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest(msgPersonNameRequired))
			return
		}
		after, before, err := svc.UpdatePerson(c.Request.Context(), mgr, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.Set(audit.CtxBefore, before)
		c.Set(audit.CtxAfter, after)
		c.JSON(http.StatusOK, after)
	}
}
func deletePerson(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		if err := svc.DeletePerson(c.Request.Context(), mgr, id); err != nil {
			writeServiceErr(c, err)
			return
		}
		c.Set(audit.CtxObjectID, id)
		c.JSON(http.StatusOK, gin.H{"deleted_at": time.Now().UTC()})
	}
}

// --- occupancies ---------------------------------------------------------------

func listOccupancies(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		unitID, ok := parseID(c)
		if !ok {
			return
		}
		items, err := svc.ListOccupancies(c.Request.Context(), mgr, unitID)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func addOccupancy(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		unitID, ok := parseID(c)
		if !ok {
			return
		}
		var in OccupancyInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest(msgRelInvalid))
			return
		}
		o, err := svc.AddOccupancy(c.Request.Context(), mgr, unitID, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		setOccupancyAuditCtx(c, unitID, o)
		c.JSON(http.StatusCreated, o)
	}
}

func endOccupancy(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		id, ok := parseID(c)
		if !ok {
			return
		}
		var in EndOccupancyInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest(msgDateRequired))
			return
		}
		o, err := svc.EndOccupancy(c.Request.Context(), mgr, id, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		setOccupancyAuditCtx(c, o.UnitID, o)
		c.JSON(http.StatusOK, o)
	}
}

// --- occupant counts -----------------------------------------------------------

func listOccupantCounts(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		unitID, ok := parseID(c)
		if !ok {
			return
		}
		items, err := svc.ListOccupantCounts(c.Request.Context(), mgr, unitID)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}

func recordOccupantCount(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		mgr, ok := requireManager(c)
		if !ok {
			return
		}
		unitID, ok := parseID(c)
		if !ok {
			return
		}
		var in OccupantCountInput
		if err := c.ShouldBindJSON(&in); err != nil {
			httpx.WriteError(c, httpx.BadRequest(msgCountInvalid))
			return
		}
		oc, err := svc.RecordOccupantCount(c.Request.Context(), mgr, unitID, in)
		if err != nil {
			writeServiceErr(c, err)
			return
		}
		c.JSON(http.StatusCreated, oc)
	}
}
