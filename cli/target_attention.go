package cli

import (
	"strings"
	"time"
)

const (
	TargetAttentionHealthy          = "healthy"
	TargetAttentionNeedsAttention   = "needs_attention"
	TargetAttentionTransportBlocked = "transport_blocked"
	TargetAttentionProgressBlocked  = "progress_blocked"
	TargetAttentionOwnershipRisky   = "ownership_risky"
	targetAttentionStaleMinutes     = 15
)

type TargetAttentionFacts struct {
	Target                string `json:"target,omitempty"`
	Kind                  string `json:"kind,omitempty"`
	InboxLastID           int64  `json:"inbox_last_id,omitempty"`
	CursorLastSeenID      int64  `json:"cursor_last_seen_id,omitempty"`
	Unread                int    `json:"unread,omitempty"`
	CursorLag             int64  `json:"cursor_lag,omitempty"`
	TransportState        string `json:"transport_state,omitempty"`
	TransportAccepted     bool   `json:"transport_accepted,omitempty"`
	LastTransportAcceptAt string `json:"last_transport_accept_at,omitempty"`
	DeliveryGraceExpired  bool   `json:"delivery_grace_expired,omitempty"`
	JournalStaleMinutes   int    `json:"journal_stale_minutes,omitempty"`
	LastOutputChangeAt    string `json:"last_output_change_at,omitempty"`
	OutputStaleMinutes    int    `json:"output_stale_minutes,omitempty"`
	PresenceState         string `json:"presence_state,omitempty"`
	RuntimeState          string `json:"runtime_state,omitempty"`
	AttentionState        string `json:"attention_state,omitempty"`
}

func BuildTargetAttentionFacts(runDir string, snapshot *ActivitySnapshot) (map[string]TargetAttentionFacts, error) {
	sessionState, err := LoadSessionsRuntimeState(SessionsRuntimeStatePath(runDir))
	if err != nil {
		return nil, err
	}
	liveness, err := LoadLivenessState(LivenessPath(runDir))
	if err != nil {
		return nil, err
	}
	return buildTargetAttentionFacts(runDir, snapshot, sessionState, liveness), nil
}

func loadTargetAttentionFacts(runDir string) map[string]TargetAttentionFacts {
	activity, err := LoadActivitySnapshot(ActivityPath(runDir))
	if err == nil && activity != nil && len(activity.Attention) > 0 {
		return activity.Attention
	}
	attention, err := BuildTargetAttentionFacts(runDir, activity)
	if err != nil {
		return nil
	}
	return attention
}

func buildTargetAttentionFacts(runDir string, snapshot *ActivitySnapshot, sessionState *SessionsRuntimeState, liveness *LivenessState) map[string]TargetAttentionFacts {
	now := time.Now().UTC()
	if snapshot != nil {
		if checkedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(snapshot.CheckedAt)); err == nil {
			now = checkedAt
		}
	}
	facts := map[string]TargetAttentionFacts{}
	transportFacts, _ := LoadTransportFacts(TransportFactsPath(runDir))

	facts["master"] = buildMasterAttentionFacts(runDir, snapshot, sessionState, liveness, transportFacts, now)

	indexes, err := existingSessionIndexes(runDir)
	if err != nil {
		return facts
	}
	for _, idx := range indexes {
		name := SessionName(idx)
		facts[name] = buildSessionAttentionFacts(runDir, name, snapshot, sessionState, liveness, transportFacts, now)
	}
	if len(facts) == 0 {
		return nil
	}
	return facts
}

func buildMasterAttentionFacts(runDir string, snapshot *ActivitySnapshot, sessionState *SessionsRuntimeState, liveness *LivenessState, transportFacts *TransportFacts, now time.Time) TargetAttentionFacts {
	inbox := readControlInboxState(MasterInboxPath(runDir), MasterCursorPath(runDir))
	transport := latestAttentionTransportFacts(transportFacts, "master")
	facts := TargetAttentionFacts{
		Target:                "master",
		Kind:                  "master",
		InboxLastID:           inbox.LastID,
		CursorLastSeenID:      inbox.LastSeenID,
		Unread:                inbox.Unread,
		CursorLag:             inbox.LastID - inbox.LastSeenID,
		TransportState:        canonicalTransportStateOrUnknown(transport.TransportState),
		TransportAccepted:     isAcceptedTUITransportState(transport.TransportState),
		LastTransportAcceptAt: transport.LastTransportAcceptAt,
		DeliveryGraceExpired:  transportAcceptExpired(transport.LastTransportAcceptAt, now),
		PresenceState:         attentionPresenceState(snapshot, "master"),
		RuntimeState:          "active",
		LastOutputChangeAt:    attentionLastOutputChangeAt(snapshot, "master"),
		OutputStaleMinutes:    attentionOutputStaleMinutes(snapshot, "master", now),
	}
	if liveness != nil {
		facts.JournalStaleMinutes = liveness.Master.JournalStaleMinutes
	}
	facts.AttentionState = deriveTargetAttentionState(facts)
	return facts
}

func buildSessionAttentionFacts(runDir, sessionName string, snapshot *ActivitySnapshot, sessionState *SessionsRuntimeState, liveness *LivenessState, transportFacts *TransportFacts, now time.Time) TargetAttentionFacts {
	inbox := readControlInboxState(ControlInboxPath(runDir, sessionName), SessionCursorPath(runDir, sessionName))
	transport := latestAttentionTransportFacts(transportFacts, sessionName)
	facts := TargetAttentionFacts{
		Target:                sessionName,
		Kind:                  "session",
		InboxLastID:           inbox.LastID,
		CursorLastSeenID:      inbox.LastSeenID,
		Unread:                inbox.Unread,
		CursorLag:             inbox.LastID - inbox.LastSeenID,
		TransportState:        canonicalTransportStateOrUnknown(transport.TransportState),
		TransportAccepted:     isAcceptedTUITransportState(transport.TransportState),
		LastTransportAcceptAt: transport.LastTransportAcceptAt,
		DeliveryGraceExpired:  transportAcceptExpired(transport.LastTransportAcceptAt, now),
		PresenceState:         attentionPresenceState(snapshot, sessionName),
		RuntimeState:          attentionRuntimeState(sessionState, sessionName),
		LastOutputChangeAt:    attentionLastOutputChangeAt(snapshot, sessionName),
		OutputStaleMinutes:    attentionOutputStaleMinutes(snapshot, sessionName, now),
	}
	if liveness != nil && liveness.Sessions != nil {
		if live, ok := liveness.Sessions[sessionName]; ok {
			facts.JournalStaleMinutes = live.JournalStaleMinutes
		}
	}
	facts.AttentionState = deriveTargetAttentionState(facts)
	return facts
}

func latestAttentionTransportFacts(all *TransportFacts, target string) TransportTargetFacts {
	if all == nil || all.Targets == nil {
		return TransportTargetFacts{}
	}
	return all.Targets[target]
}

func attentionPresenceState(snapshot *ActivitySnapshot, target string) string {
	if snapshot == nil || snapshot.Targets == nil {
		return ""
	}
	return strings.TrimSpace(snapshot.Targets[target].State)
}

func attentionRuntimeState(sessionState *SessionsRuntimeState, target string) string {
	if sessionState == nil || sessionState.Sessions == nil {
		return ""
	}
	return strings.TrimSpace(sessionState.Sessions[target].State)
}

func attentionLastOutputChangeAt(snapshot *ActivitySnapshot, target string) string {
	if snapshot == nil {
		return ""
	}
	if target == "master" {
		if actor, ok := snapshot.Actors["master"]; ok {
			return strings.TrimSpace(actor.LastOutputChangeAt)
		}
		return ""
	}
	if snapshot.Sessions == nil {
		return ""
	}
	return strings.TrimSpace(snapshot.Sessions[target].LastOutputChangeAt)
}

func attentionOutputStaleMinutes(snapshot *ActivitySnapshot, target string, now time.Time) int {
	return staleMinutesSince(attentionLastOutputChangeAt(snapshot, target), now)
}

func staleMinutesSince(ts string, now time.Time) int {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(ts))
	if err != nil || parsed.IsZero() || parsed.After(now) {
		return 0
	}
	return int(now.Sub(parsed).Minutes())
}

func transportAcceptExpired(ts string, now time.Time) bool {
	if staleMinutesSince(ts, now) < targetAttentionStaleMinutes {
		return false
	}
	return strings.TrimSpace(ts) != ""
}

func deriveTargetAttentionState(facts TargetAttentionFacts) string {
	presence := strings.TrimSpace(facts.PresenceState)
	runtimeState := strings.TrimSpace(facts.RuntimeState)
	transportState := normalizeTUITransportState(facts.TransportState)

	switch {
	case presence != "" && presence != TargetPresencePresent && presence != TargetPresenceUnknown:
		return TargetAttentionOwnershipRisky
	case runtimeState == "parked" || runtimeState == "done" || runtimeState == "blocked":
		return TargetAttentionOwnershipRisky
	case facts.Unread > 0 && facts.CursorLag > 0:
		if facts.DeliveryGraceExpired || facts.JournalStaleMinutes >= targetAttentionStaleMinutes || transportState == TUIStateInterrupted || transportState == TUIStateProviderDialog || transportState == TUIStateBufferedInput {
			return TargetAttentionTransportBlocked
		}
		return TargetAttentionNeedsAttention
	case facts.JournalStaleMinutes >= targetAttentionStaleMinutes || facts.OutputStaleMinutes >= targetAttentionStaleMinutes:
		if runtimeState == "" || runtimeState == "active" || runtimeState == "progress" || runtimeState == "working" {
			return TargetAttentionProgressBlocked
		}
		return TargetAttentionNeedsAttention
	case facts.Unread > 0 || facts.CursorLag > 0:
		return TargetAttentionNeedsAttention
	default:
		return TargetAttentionHealthy
	}
}

func targetAttentionBlocked(attention TargetAttentionFacts) bool {
	switch strings.TrimSpace(attention.AttentionState) {
	case TargetAttentionTransportBlocked, TargetAttentionProgressBlocked:
		return true
	default:
		return false
	}
}

func targetAttentionNeedsAction(attention TargetAttentionFacts) bool {
	switch strings.TrimSpace(attention.AttentionState) {
	case TargetAttentionNeedsAttention, TargetAttentionTransportBlocked, TargetAttentionProgressBlocked, TargetAttentionOwnershipRisky:
		return true
	default:
		return false
	}
}
