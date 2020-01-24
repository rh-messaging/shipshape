package entities

import (
	"encoding/json"
	"github.com/rh-messaging/shipshape/pkg/apps/qdrouterd/qdrmanagement/entities/common"
	"strconv"
	"strings"
)

// Link represents a Dispatch Router router.link entity
type Link struct {
	AdminStatus            AdminStatusType      `json:"adminStatus,string"`
	OperStatus             LinkOperStatusType   `json:"operStatus,string"`
	LinkName               string               `json:"linkName"`
	LinkType               string               `json:"linkType"`
	LinkDir                common.DirectionType `json:"linkDir,string"`
	OwningAddr             string               `json:"owningAddr"`
	Capacity               int                  `json:"capacity"`
	Peer                   string               `json:"peer"`
	UndeliveredCount       int                  `json:"undeliveredCount"`
	UnsettledCount         int                  `json:"unsettledCount"`
	DeliveryCount          int                  `json:"deliveryCount"`
	PresettledCount        int                  `json:"presettledCount"`
	DroppedPresettledCount int                  `json:"droppedPresettledCount"`
	AcceptedCount          int                  `json:"acceptedCount"`
	RejectedCount          int                  `json:"rejectedCount"`
	ReleasedCount          int                  `json:"releasedCount"`
	ModifiedCount          int                  `json:"modifiedCount"`
	DeliveriesDelayed1Sec  int                  `json:"deliveriesDelayed1Sec"`
	DeliveriesDelayed10Sec int                  `json:"deliveriesDelayed10Sec"`
	DeliveriesStuck        int                  `json:"deliveriesStuck"`
	CreditAvailable        int                  `json:"creditAvailable"`
	ZeroCreditSeconds      int                  `json:"zeroCreditSeconds"`
	SettleRate             int                  `json:"settleRate"`
	IngressHistogram       []int                `json:"ingressHistogram"`
	Priority               int                  `json:"priority"`
}

// Implementation of the Entity interface
func (Link) GetEntityId() string {
	return "link"
}

type LinkOperStatusType int

const (
	LinkOperStatusUp LinkOperStatusType = iota
	LinkOperStatusDown
	LinkOperStatusQuiescing
	LinkOperStatusIdle
)

// UnmarshalJSON
func (l *LinkOperStatusType) UnmarshalJSON(b []byte) error {
	var s string

	if len(b) == 0 {
		return nil
	}
	if b[0] != '"' {
		b = []byte(strconv.Quote(string(b)))
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	switch strings.ToLower(s) {
	case "up":
		*l = LinkOperStatusUp
	case "down":
		*l = LinkOperStatusDown
	case "quiescing":
		*l = LinkOperStatusQuiescing
	case "idle":
		*l = LinkOperStatusIdle
	}

	return nil
}

// MarshalJSON
func (l LinkOperStatusType) MarshalJSON() ([]byte, error) {
	var s string
	switch l {
	case LinkOperStatusUp:
		s = "up"
	case LinkOperStatusDown:
		s = "down"
	case LinkOperStatusQuiescing:
		s = "quiescing"
	case LinkOperStatusIdle:
		s = "idle"
	}
	return json.Marshal(s)
}
