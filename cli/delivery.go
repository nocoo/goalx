package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type TransportDeliveryOutcome struct {
	SubmitMode     string
	TransportState string
}

type TransportDeliverFunc func(target, engine string) (TransportDeliveryOutcome, error)

func latestSessionDelivery(runDir, sessionName string) (ControlDelivery, bool) {
	deliveries, err := LoadControlDeliveries(ControlDeliveriesPath(runDir))
	if err != nil || deliveries == nil {
		return ControlDelivery{}, false
	}
	prefix := "session-inbox:" + sessionName + ":"
	dedupe := "session-wake:" + sessionName
	var latest ControlDelivery
	found := false
	for _, item := range deliveries.Items {
		if !strings.HasPrefix(item.DedupeKey, prefix) && item.DedupeKey != dedupe {
			continue
		}
		if !found || item.AttemptedAt > latest.AttemptedAt {
			latest = item
			found = true
		}
	}
	return latest, found
}

func latestSessionInboxDelivery(runDir, sessionName string) (ControlDelivery, bool) {
	deliveries, err := LoadControlDeliveries(ControlDeliveriesPath(runDir))
	if err != nil || deliveries == nil {
		return ControlDelivery{}, false
	}
	prefix := "session-inbox:" + sessionName + ":"
	var latest ControlDelivery
	found := false
	for _, item := range deliveries.Items {
		if !strings.HasPrefix(item.DedupeKey, prefix) {
			continue
		}
		if !found || item.AttemptedAt > latest.AttemptedAt {
			latest = item
			found = true
		}
	}
	return latest, found
}

func latestTargetDelivery(runDir, logicalTarget string) (ControlDelivery, bool) {
	logicalTarget = strings.TrimSpace(logicalTarget)
	if logicalTarget == "" {
		return ControlDelivery{}, false
	}
	if logicalTarget == "master" {
		deliveries, err := LoadControlDeliveries(ControlDeliveriesPath(runDir))
		if err != nil || deliveries == nil {
			return ControlDelivery{}, false
		}
		var latest ControlDelivery
		found := false
		for _, item := range deliveries.Items {
			if !strings.HasPrefix(item.DedupeKey, "master-") && !strings.HasPrefix(item.DedupeKey, "session-") {
				continue
			}
			if !strings.Contains(item.Target, ":master") {
				continue
			}
			if !found || item.AttemptedAt > latest.AttemptedAt {
				latest = item
				found = true
			}
		}
		return latest, found
	}
	return latestSessionDelivery(runDir, logicalTarget)
}

func deliveryAcceptedWithin(delivery ControlDelivery, window time.Duration, now time.Time) bool {
	return deliveryTimestampWithin(delivery.AcceptedAt, window, now)
}

func deliveryTimestampWithin(ts string, window time.Duration, now time.Time) bool {
	if window <= 0 || strings.TrimSpace(ts) == "" {
		return false
	}
	at, err := time.Parse(time.RFC3339, strings.TrimSpace(ts))
	if err != nil {
		return false
	}
	return now.Sub(at) < window
}

func DeliverControlNudge(runDir, messageID, dedupeKey, target, engine string, deliver TransportDeliverFunc) (*ControlDelivery, error) {
	return deliverControlNudge(runDir, messageID, dedupeKey, target, engine, true, deliver)
}

func reconcileControlDeliveries(runDir string) error {
	deliveries, err := LoadControlDeliveries(ControlDeliveriesPath(runDir))
	if err != nil || deliveries == nil {
		return err
	}
	updated := false
	if cursor, err := LoadMasterCursorState(MasterCursorPath(runDir)); err == nil {
		if reconcileInboxDeliveryAcceptance(deliveries, "master", cursor) {
			updated = true
		}
	}
	indexes, err := existingSessionIndexes(runDir)
	if err != nil {
		return err
	}
	for _, idx := range indexes {
		sessionName := SessionName(idx)
		cursor, err := LoadMasterCursorState(SessionCursorPath(runDir, sessionName))
		if err != nil {
			continue
		}
		if reconcileInboxDeliveryAcceptance(deliveries, sessionName, cursor) {
			updated = true
		}
	}
	if !updated {
		return nil
	}
	return SaveControlDeliveries(ControlDeliveriesPath(runDir), deliveries)
}

func reconcileTargetDeliveries(runDir, target string, cursor *MasterCursorState) error {
	deliveries, err := LoadControlDeliveries(ControlDeliveriesPath(runDir))
	if err != nil || deliveries == nil {
		return err
	}
	if !reconcileInboxDeliveryAcceptance(deliveries, target, cursor) {
		return nil
	}
	return SaveControlDeliveries(ControlDeliveriesPath(runDir), deliveries)
}

func reconcileInboxDeliveryAcceptance(deliveries *ControlDeliveries, target string, cursor *MasterCursorState) bool {
	if deliveries == nil || cursor == nil || cursor.LastSeenID <= 0 {
		return false
	}
	acceptedAt := strings.TrimSpace(cursor.UpdatedAt)
	if acceptedAt == "" {
		acceptedAt = time.Now().UTC().Format(time.RFC3339)
	}
	updated := false
	for i := range deliveries.Items {
		item := &deliveries.Items[i]
		if item.AcceptedAt != "" {
			continue
		}
		switch strings.TrimSpace(item.Status) {
		case "failed", "cancelled":
			continue
		}
		messageID, ok := controlInboxDeliveryMessageID(target, item.MessageID)
		if !ok {
			messageID, ok = controlInboxDeliveryMessageID(target, item.DedupeKey)
		}
		if !ok || messageID > cursor.LastSeenID {
			continue
		}
		item.Status = "accepted"
		item.LastError = ""
		item.AcceptedAt = acceptedAt
		if strings.TrimSpace(item.AttemptedAt) == "" {
			item.AttemptedAt = acceptedAt
		}
		updated = true
	}
	return updated
}

func controlInboxDeliveryMessageID(target, raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	prefix := "master-inbox:"
	if target != "master" {
		prefix = "session-inbox:" + target + ":"
	}
	if !strings.HasPrefix(raw, prefix) {
		return 0, false
	}
	id, err := strconv.ParseInt(strings.TrimPrefix(raw, prefix), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func deliverControlNudge(runDir, messageID, dedupeKey, target, engine string, dedupeOnSuccess bool, deliver TransportDeliverFunc) (*ControlDelivery, error) {
	if err := EnsureControlState(runDir); err != nil {
		return nil, err
	}
	dedupeKey = strings.TrimSpace(dedupeKey)
	if dedupeKey == "" {
		dedupeKey = strings.TrimSpace(messageID)
	}
	if dedupeKey == "" {
		return nil, fmt.Errorf("delivery dedupe key is required")
	}
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		messageID = dedupeKey
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("delivery target is required")
	}

	deliveries, err := LoadControlDeliveries(ControlDeliveriesPath(runDir))
	if err != nil {
		return nil, err
	}

	idx := -1
	for i := range deliveries.Items {
		if deliveries.Items[i].DedupeKey == dedupeKey {
			idx = i
			break
		}
	}
	if idx == -1 {
		deliveries.Items = append(deliveries.Items, ControlDelivery{
			DeliveryID: newControlObjectID("delivery"),
			DedupeKey:  dedupeKey,
		})
		idx = len(deliveries.Items) - 1
	}

	item := &deliveries.Items[idx]
	item.MessageID = messageID
	item.DedupeKey = dedupeKey
	item.Target = target
	item.Adapter = "tmux"
	if item.DeliveryID == "" {
		item.DeliveryID = newControlObjectID("delivery")
	}
	if dedupeOnSuccess && item.Status == "accepted" && item.AcceptedAt != "" {
		if err := SaveControlDeliveries(ControlDeliveriesPath(runDir), deliveries); err != nil {
			return nil, err
		}
		copy := *item
		return &copy, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	item.AttemptedAt = now
	item.Status = "pending"
	item.SubmitMode = ""
	item.TransportState = ""
	item.LastError = ""
	if err := SaveControlDeliveries(ControlDeliveriesPath(runDir), deliveries); err != nil {
		return nil, err
	}

	outcome := TransportDeliveryOutcome{}
	if deliver != nil {
		result, err := deliver(target, engine)
		outcome = result
		if err != nil {
			item.Status = "failed"
			item.SubmitMode = outcome.SubmitMode
			item.TransportState = ""
			item.LastError = err.Error()
			item.AcceptedAt = ""
			if saveErr := SaveControlDeliveries(ControlDeliveriesPath(runDir), deliveries); saveErr != nil {
				return nil, saveErr
			}
			copy := *item
			return &copy, err
		}
	}

	item.SubmitMode = outcome.SubmitMode
	item.TransportState = strings.TrimSpace(outcome.TransportState)
	if item.TransportState == "" {
		item.TransportState = string(TUIStateUnknown)
	}
	item.Status = item.TransportState
	if isAcceptedTUITransportState(item.TransportState) {
		item.Status = "accepted"
	}
	item.LastError = ""
	item.AcceptedAt = ""
	if item.Status == "accepted" {
		item.AcceptedAt = now
	}
	if err := SaveControlDeliveries(ControlDeliveriesPath(runDir), deliveries); err != nil {
		return nil, err
	}
	copy := *item
	return &copy, nil
}

func newControlObjectID(prefix string) string {
	id := newRunID()
	if strings.HasPrefix(id, runIDPrefix) {
		return prefix + id[len(runIDPrefix):]
	}
	return prefix + "_" + id
}
