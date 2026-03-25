package cli

import "time"

func QueueControlReminder(runDir, dedupeKey, reason, target string) (*ControlReminder, error) {
	return QueueControlReminderWithEngine(runDir, dedupeKey, reason, target, "")
}

func QueueControlReminderWithEngine(runDir, dedupeKey, reason, target, engine string) (*ControlReminder, error) {
	if err := EnsureControlState(runDir); err != nil {
		return nil, err
	}
	reminders, err := LoadControlReminders(ControlRemindersPath(runDir))
	if err != nil {
		return nil, err
	}
	for i := range reminders.Items {
		item := &reminders.Items[i]
		if item.DedupeKey != dedupeKey {
			continue
		}
		changed := false
		if item.Reason != reason {
			item.Reason = reason
			changed = true
		}
		if item.Target != target {
			item.Target = target
			changed = true
		}
		if item.Engine != engine {
			item.Engine = engine
			changed = true
		}
		if item.Suppressed || item.ResolvedAt != "" {
			item.Suppressed = false
			item.ResolvedAt = ""
			item.CooldownUntil = ""
			item.Attempts = 0
			changed = true
		}
		if changed {
			if err := SaveControlReminders(ControlRemindersPath(runDir), reminders); err != nil {
				return nil, err
			}
		}
		if !item.Suppressed && item.ResolvedAt == "" {
			copy := *item
			return &copy, nil
		}
	}
	item := ControlReminder{
		ReminderID: newControlObjectID("reminder"),
		DedupeKey:  dedupeKey,
		Reason:     reason,
		Target:     target,
		Engine:     engine,
	}
	reminders.Items = append(reminders.Items, item)
	if err := SaveControlReminders(ControlRemindersPath(runDir), reminders); err != nil {
		return nil, err
	}
	copy := item
	return &copy, nil
}

func SuppressControlReminder(runDir, dedupeKey string) error {
	if err := EnsureControlState(runDir); err != nil {
		return err
	}
	reminders, err := LoadControlReminders(ControlRemindersPath(runDir))
	if err != nil {
		return err
	}
	changed := false
	for i := range reminders.Items {
		item := &reminders.Items[i]
		if item.DedupeKey != dedupeKey || item.Suppressed {
			continue
		}
		item.Suppressed = true
		changed = true
	}
	if !changed {
		return nil
	}
	return SaveControlReminders(ControlRemindersPath(runDir), reminders)
}

func DeliverDueControlReminders(runDir, engine string, interval time.Duration, deliver TransportDeliverFunc) error {
	if err := EnsureControlState(runDir); err != nil {
		return err
	}
	reminders, err := LoadControlReminders(ControlRemindersPath(runDir))
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	changed := false
	for i := range reminders.Items {
		item := &reminders.Items[i]
		if item.Suppressed || item.ResolvedAt != "" {
			continue
		}
		if item.CooldownUntil != "" {
			cooldownUntil, err := time.Parse(time.RFC3339, item.CooldownUntil)
			if err == nil && cooldownUntil.After(now) {
				continue
			}
		}
		deliveryEngine := item.Engine
		if deliveryEngine == "" {
			deliveryEngine = engine
		}
		delivery, _ := deliverControlNudge(runDir, item.ReminderID, item.DedupeKey, item.Target, deliveryEngine, false, deliver)
		item.Attempts++
		item.CooldownUntil = now.Add(controlReminderCooldown(interval, item.Attempts, delivery)).Format(time.RFC3339)
		changed = true
	}
	if !changed {
		return nil
	}
	return SaveControlReminders(ControlRemindersPath(runDir), reminders)
}

func controlReminderCooldown(interval time.Duration, attempts int, delivery *ControlDelivery) time.Duration {
	base := interval
	if base < time.Minute {
		base = time.Minute
	}
	if delivery == nil {
		return base
	}
	switch delivery.Status {
	case "buffered":
		cooldown := interval / 4
		if cooldown < 5*time.Second {
			cooldown = 5 * time.Second
		}
		if cooldown > 30*time.Second {
			cooldown = 30 * time.Second
		}
		return cooldown
	case "sent":
		return base
	case "failed":
		multiplier := attempts
		if multiplier < 1 {
			multiplier = 1
		}
		if multiplier > 4 {
			multiplier = 4
		}
		return time.Duration(multiplier) * base
	}
	return base
}
