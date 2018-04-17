package api

import (
	"fmt"

	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/bus"
	m "github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/alerting"
	"github.com/grafana/grafana/pkg/services/guardian"
)

func ValidateOrgAlert(c *m.ReqContext) {
	id := c.ParamsInt64(":alertId")
	query := m.GetAlertByIdQuery{Id: id}

	if err := bus.Dispatch(&query); err != nil {
		c.JsonApiErr(404, "Warnung nicht gefunden", nil)
		return
	}

	if c.OrgId != query.Result.OrgId {
		c.JsonApiErr(403, "Sie dürfen keine Warnung bearbeiten/anzeigen", nil)
		return
	}
}

func GetAlertStatesForDashboard(c *m.ReqContext) Response {
	dashboardID := c.QueryInt64("dashboardId")

	if dashboardID == 0 {
		return Error(400, "Fehlender Abfrageparameter dashboardId", nil)
	}

	query := m.GetAlertStatesForDashboardQuery{
		OrgId:       c.OrgId,
		DashboardId: c.QueryInt64("dashboardId"),
	}

	if err := bus.Dispatch(&query); err != nil {
		return Error(500, "Fehler beim Abrufen von Warnungszuständen", err)
	}

	return JSON(200, query.Result)
}

// GET /api/alerts
func GetAlerts(c *m.ReqContext) Response {
	query := m.GetAlertsQuery{
		OrgId:       c.OrgId,
		DashboardId: c.QueryInt64("dashboardId"),
		PanelId:     c.QueryInt64("panelId"),
		Limit:       c.QueryInt64("limit"),
		User:        c.SignedInUser,
	}

	states := c.QueryStrings("state")
	if len(states) > 0 {
		query.State = states
	}

	if err := bus.Dispatch(&query); err != nil {
		return Error(500, "List alerts failed", err)
	}

	for _, alert := range query.Result {
		alert.Url = m.GetDashboardUrl(alert.DashboardUid, alert.DashboardSlug)
	}

	return JSON(200, query.Result)
}

// POST /api/alerts/test
func AlertTest(c *m.ReqContext, dto dtos.AlertTestCommand) Response {
	if _, idErr := dto.Dashboard.Get("id").Int64(); idErr != nil {
		return Error(400, "Das Dashboard muss mindestens einmal gespeichert werden, bevor Sie eine Warnungsregel testen können", nil)
	}

	backendCmd := alerting.AlertTestCommand{
		OrgId:     c.OrgId,
		Dashboard: dto.Dashboard,
		PanelId:   dto.PanelId,
	}

	if err := bus.Dispatch(&backendCmd); err != nil {
		if validationErr, ok := err.(alerting.ValidationError); ok {
			return Error(422, validationErr.Error(), nil)
		}
		return Error(500, "Testen der Regel ist fehlgeschlagen", err)
	}

	res := backendCmd.Result
	dtoRes := &dtos.AlertTestResult{
		Firing:         res.Firing,
		ConditionEvals: res.ConditionEvals,
		State:          res.Rule.State,
	}

	if res.Error != nil {
		dtoRes.Error = res.Error.Error()
	}

	for _, log := range res.Logs {
		dtoRes.Logs = append(dtoRes.Logs, &dtos.AlertTestResultLog{Message: log.Message, Data: log.Data})
	}
	for _, match := range res.EvalMatches {
		dtoRes.EvalMatches = append(dtoRes.EvalMatches, &dtos.EvalMatch{Metric: match.Metric, Value: match.Value})
	}

	dtoRes.TimeMs = fmt.Sprintf("%1.3fms", res.GetDurationMs())

	return JSON(200, dtoRes)
}

// GET /api/alerts/:id
func GetAlert(c *m.ReqContext) Response {
	id := c.ParamsInt64(":alertId")
	query := m.GetAlertByIdQuery{Id: id}

	if err := bus.Dispatch(&query); err != nil {
		return Error(500, "Listenalarme fehlgeschlagen", err)
	}

	return JSON(200, &query.Result)
}

func GetAlertNotifiers(c *m.ReqContext) Response {
	return JSON(200, alerting.GetNotifiers())
}

func GetAlertNotifications(c *m.ReqContext) Response {
	query := &m.GetAllAlertNotificationsQuery{OrgId: c.OrgId}

	if err := bus.Dispatch(query); err != nil {
		return Error(500, "Fehler beim Abrufen von Warnmeldungen", err)
	}

	result := make([]*dtos.AlertNotification, 0)

	for _, notification := range query.Result {
		result = append(result, &dtos.AlertNotification{
			Id:        notification.Id,
			Name:      notification.Name,
			Type:      notification.Type,
			IsDefault: notification.IsDefault,
			Created:   notification.Created,
			Updated:   notification.Updated,
		})
	}

	return JSON(200, result)
}

func GetAlertNotificationByID(c *m.ReqContext) Response {
	query := &m.GetAlertNotificationsQuery{
		OrgId: c.OrgId,
		Id:    c.ParamsInt64("notificationId"),
	}

	if err := bus.Dispatch(query); err != nil {
		return Error(500, "Fehler beim Abrufen von Warnmeldung", err)
	}

	return JSON(200, query.Result)
}

func CreateAlertNotification(c *m.ReqContext, cmd m.CreateAlertNotificationCommand) Response {
	cmd.OrgId = c.OrgId

	if err := bus.Dispatch(&cmd); err != nil {
		return Error(500, "Erstellen von Warnmeldung fehlgeschlagen", err)
	}

	return JSON(200, cmd.Result)
}

func UpdateAlertNotification(c *m.ReqContext, cmd m.UpdateAlertNotificationCommand) Response {
	cmd.OrgId = c.OrgId

	if err := bus.Dispatch(&cmd); err != nil {
		return Error(500, "Aktualisierung von Warnmeldung Fehlgeschlagen", err)
	}

	return JSON(200, cmd.Result)
}

func DeleteAlertNotification(c *m.ReqContext) Response {
	cmd := m.DeleteAlertNotificationCommand{
		OrgId: c.OrgId,
		Id:    c.ParamsInt64("notificationId"),
	}

	if err := bus.Dispatch(&cmd); err != nil {
		return Error(500, "Löschen von Warnmeldung fehlgeschlagen", err)
	}

	return Success("Benachrichtigung gelöscht")
}

//POST /api/alert-notifications/test
func NotificationTest(c *m.ReqContext, dto dtos.NotificationTestCommand) Response {
	cmd := &alerting.NotificationTestCommand{
		Name:     dto.Name,
		Type:     dto.Type,
		Settings: dto.Settings,
	}

	if err := bus.Dispatch(cmd); err != nil {
		if err == m.ErrSmtpNotEnabled {
			return Error(412, err.Error(), err)
		}
		return Error(500, "Fehler beim Senden von Warnmeldungen", err)
	}

	return Success("Testbenachrichtigung gesendet")
}

//POST /api/alerts/:alertId/pause
func PauseAlert(c *m.ReqContext, dto dtos.PauseAlertCommand) Response {
	alertID := c.ParamsInt64("alertId")

	query := m.GetAlertByIdQuery{Id: alertID}

	if err := bus.Dispatch(&query); err != nil {
		return Error(500, "Get Alert failed", err)
	}

	guardian := guardian.New(query.Result.DashboardId, c.OrgId, c.SignedInUser)
	if canEdit, err := guardian.CanEdit(); err != nil || !canEdit {
		if err != nil {
			return Error(500, "Fehler beim überprüfen der Berechtigungen für die Warnung", err)
		}

		return Error(403, "Zugriff auf dieses Dashboard und diese Warnung verweigert", nil)
	}

	cmd := m.PauseAlertCommand{
		OrgId:    c.OrgId,
		AlertIds: []int64{alertID},
		Paused:   dto.Paused,
	}

	if err := bus.Dispatch(&cmd); err != nil {
		return Error(500, "", err)
	}

	var response m.AlertStateType = m.AlertStatePending
	pausedState := "un-paused"
	if cmd.Paused {
		response = m.AlertStatePaused
		pausedState = "paused"
	}

	result := map[string]interface{}{
		"alertId": alertID,
		"state":   response,
		"message": "Meldung " + pausedState,
	}

	return JSON(200, result)
}

//POST /api/admin/pause-all-alerts
func PauseAllAlerts(c *m.ReqContext, dto dtos.PauseAllAlertsCommand) Response {
	updateCmd := m.PauseAllAlertCommand{
		Paused: dto.Paused,
	}

	if err := bus.Dispatch(&updateCmd); err != nil {
		return Error(500, "Warnungen konnten nicht angehalten werden", err)
	}

	var response m.AlertStateType = m.AlertStatePending
	pausedState := "un paused"
	if updateCmd.Paused {
		response = m.AlertStatePaused
		pausedState = "paused"
	}

	result := map[string]interface{}{
		"state":          response,
		"message":        "Meldungen " + pausedState,
		"alertsAffected": updateCmd.ResultCount,
	}

	return JSON(200, result)
}
