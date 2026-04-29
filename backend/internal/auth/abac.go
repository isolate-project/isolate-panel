package auth

import (
	"fmt"
	"net"
	"time"
)

// ABACPolicy defines attribute-based access control rules.
// Each policy is evaluated at request time against the request context.
type ABACPolicy struct {
	RequiredPermissions []Permission
	TimeWindow          *TimeWindowPolicy
	IPRanges            []string
	RequireMFA          bool
}

type TimeWindowPolicy struct {
	StartHour int // 0-23
	EndHour   int // 0-23
	Timezone  string
}

func (p *ABACPolicy) Evaluate(reqCtx RequestContext, perms Permissions) error {
	for _, perm := range p.RequiredPermissions {
		if !perms.Has(perm) {
			return fmt.Errorf("missing permission %s", perm)
		}
	}

	if p.TimeWindow != nil {
		loc, err := time.LoadLocation(p.TimeWindow.Timezone)
		if err != nil {
			loc = time.UTC
		}
		now := reqCtx.Timestamp.In(loc)
		if now.Hour() < p.TimeWindow.StartHour || now.Hour() > p.TimeWindow.EndHour {
			return fmt.Errorf("access denied outside allowed hours %d-%d %s",
				p.TimeWindow.StartHour, p.TimeWindow.EndHour, p.TimeWindow.Timezone)
		}
	}

	if len(p.IPRanges) > 0 {
		clientIP := net.ParseIP(reqCtx.IPAddress)
		if clientIP == nil {
			return fmt.Errorf("invalid client IP")
		}
		allowed := false
		for _, r := range p.IPRanges {
			_, ipNet, err := net.ParseCIDR(r)
			if err != nil {
				ip := net.ParseIP(r)
				if ip != nil && ip.Equal(clientIP) {
					allowed = true
					break
				}
				continue
			}
			if ipNet.Contains(clientIP) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("access denied from IP %s", reqCtx.IPAddress)
		}
	}

	if p.RequireMFA && !reqCtx.MFAConfirmed {
		return fmt.Errorf("MFA required")
	}

	return nil
}

// RequestContext captures runtime attributes for ABAC evaluation.
type RequestContext struct {
	Timestamp     time.Time
	IPAddress     string
	UserAgent     string
	MFAConfirmed  bool
	ResourceID    uint
	ResourceType  string
	Action        string
}
