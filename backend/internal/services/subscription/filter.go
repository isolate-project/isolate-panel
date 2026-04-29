package subscription

import (
	"fmt"
	"hash/fnv"
	"regexp"

	"github.com/isolate-project/isolate-panel/internal/logger"
	"github.com/isolate-project/isolate-panel/internal/models"
)

// SubscriptionFilter defines filtering criteria for subscription inbounds
type SubscriptionFilter struct {
	ProtocolRegex string // regex to match inbound.Protocol (e.g., "vless|vmess|trojan")
	CoreName      string // exact match on inbound.Core.Name (e.g., "xray", "singbox", "mihomo")
	CoreNameRegex string // regex to match inbound.Core.Name
	TagRegex      string // regex to match inbound.Name (e.g., "US*", ".*-ws")
}

// FilterInbounds filters inbounds based on the filter criteria
func (f *SubscriptionFilter) FilterInbounds(inbounds []models.Inbound) []models.Inbound {
	if f == nil || (f.ProtocolRegex == "" && f.CoreName == "" && f.CoreNameRegex == "" && f.TagRegex == "") {
		return inbounds
	}

	var result []models.Inbound
	for _, inbound := range inbounds {
		// Apply ProtocolRegex filter
		if f.ProtocolRegex != "" {
			re, err := regexp.Compile(f.ProtocolRegex)
			if err != nil {
				// Invalid regex, log warning and skip this filter
				logger.Log.Warn().Err(err).Str("regex", f.ProtocolRegex).Msg("Invalid protocol regex, skipping filter")
			} else if !re.MatchString(inbound.Protocol) {
				continue
			}
		}

		// Apply CoreName exact match filter
		if f.CoreName != "" {
			if inbound.Core == nil || inbound.Core.Name != f.CoreName {
				continue
			}
		}

		// Apply CoreNameRegex filter
		if f.CoreNameRegex != "" {
			if inbound.Core == nil {
				continue
			}
			re, err := regexp.Compile(f.CoreNameRegex)
			if err != nil {
				// Invalid regex, log warning and skip this filter
				logger.Log.Warn().Err(err).Str("regex", f.CoreNameRegex).Msg("Invalid core name regex, skipping filter")
			} else if !re.MatchString(inbound.Core.Name) {
				continue
			}
		}

		// Apply TagRegex filter
		if f.TagRegex != "" {
			re, err := regexp.Compile(f.TagRegex)
			if err != nil {
				// Invalid regex, log warning and skip this filter
				logger.Log.Warn().Err(err).Str("regex", f.TagRegex).Msg("Invalid tag regex, skipping filter")
			} else if !re.MatchString(inbound.Name) {
				continue
			}
		}

		result = append(result, inbound)
	}

	return result
}

// Hash returns a hash of the filter criteria for caching purposes
func (f *SubscriptionFilter) Hash() string {
	if f == nil {
		return ""
	}
	h := fnv.New32a()
	fmt.Fprintf(h, "p:%s c:%s cr:%s t:%s", f.ProtocolRegex, f.CoreName, f.CoreNameRegex, f.TagRegex)
	return fmt.Sprintf("%08x", h.Sum32())
}
