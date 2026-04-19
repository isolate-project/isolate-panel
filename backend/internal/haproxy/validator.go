package haproxy

import (
	"fmt"
	"strings"

	"github.com/isolate-project/isolate-panel/internal/models"
	"gorm.io/gorm"
)

type ValidationSeverity string

const (
	SeverityInfo    ValidationSeverity = "info"
	SeverityWarning ValidationSeverity = "warning"
	SeverityError   ValidationSeverity = "error"
)

type ValidationAction string

const (
	ActionAllow   ValidationAction = "allow"
	ActionConfirm ValidationAction = "confirm"
	ActionBlock   ValidationAction = "block"
)

type PortConflict struct {
	InboundID         uint   `json:"inbound_id"`
	InboundName       string `json:"inbound_name"`
	Protocol          string `json:"protocol"`
	Transport         string `json:"transport,omitempty"`
	CoreType          string `json:"core_type"`
	Port              int    `json:"port"`
	HaproxyCompatible bool   `json:"haproxy_compatible"`
	CanShare          bool   `json:"can_share"`
	SharingMechanism  string `json:"sharing_mechanism,omitempty"`
	RequiresConfirm   bool   `json:"requires_confirm"`
}

type PortConflictCheck struct {
	Port              int                `json:"port"`
	ListenAddress     string             `json:"listen_address"`
	Protocol          string             `json:"protocol"`
	Transport         string             `json:"transport,omitempty"`
	CoreType          string             `json:"core_type"`
	IsAvailable       bool               `json:"is_available"`
	HaproxyCompatible bool               `json:"haproxy_compatible"`
	CanSharePort      bool               `json:"can_share_port"`
	SharingMechanism  string             `json:"sharing_mechanism,omitempty"`
	Severity          ValidationSeverity `json:"severity"`
	Action            ValidationAction   `json:"action"`
	Message           string             `json:"message"`
	Conflicts         []PortConflict     `json:"conflicts,omitempty"`
}

type PortValidator struct {
	db *gorm.DB
}

func NewPortValidator(db *gorm.DB) *PortValidator {
	return &PortValidator{db: db}
}

func (v *PortValidator) ValidatePortConflict(
	port int,
	listenAddr string,
	protocol string,
	transport string,
	coreType string,
	existingInbounds []models.Inbound,
) *PortConflictCheck {
	result := &PortConflictCheck{
		Port:          port,
		ListenAddress: listenAddr,
		Protocol:      protocol,
		Transport:     transport,
		CoreType:      coreType,
	}

	if port < 1 || port > 65535 {
		result.Severity = SeverityError
		result.Action = ActionBlock
		result.Message = "Порт должен быть от 1 до 65535"
		result.IsAvailable = false
		return result
	}

	result.HaproxyCompatible = !IsUDPTransport(transport)

	for _, existing := range existingInbounds {
		if !v.isPortOverlap(port, listenAddr, existing.Port, existing.ListenAddress) {
			continue
		}

		// Extract transport from config JSON if available
		existingTransport := v.extractTransportFromConfig(existing.ConfigJSON)
		existingCoreType := v.extractCoreTypeFromConfig(existing.ConfigJSON, existing.CoreID)

		conflict := PortConflict{
			InboundID:   existing.ID,
			InboundName: existing.Name,
			Protocol:    existing.Protocol,
			Port:        existing.Port,
			CoreType:    existingCoreType,
			Transport:   existingTransport,
		}

		existingCompatible := !IsUDPTransport(existingTransport)
		conflict.HaproxyCompatible = existingCompatible

		if result.HaproxyCompatible && existingCompatible {
			conflict.CanShare = true
			result.CanSharePort = true

			if SupportsSNI(protocol) && SupportsSNI(conflict.Protocol) {
				conflict.SharingMechanism = "sni"
				result.SharingMechanism = "sni"
			} else if SupportsPath(transport) && SupportsPath(conflict.Transport) {
				conflict.SharingMechanism = "path"
				result.SharingMechanism = "path"
			}
		} else {
			conflict.CanShare = false
		}

		if conflict.CanShare && protocol != conflict.Protocol {
			conflict.RequiresConfirm = true
		}

		result.Conflicts = append(result.Conflicts, conflict)
	}

	result.IsAvailable = len(result.Conflicts) == 0 || result.CanSharePort

	if len(result.Conflicts) == 0 {
		result.Severity = SeverityInfo
		result.Message = "✓ Порт свободен. Можно создавать инбаунд."
		result.Action = ActionAllow
	} else if result.CanSharePort {
		if result.SharingMechanism == "sni" {
			result.Severity = SeverityInfo
			result.Message = fmt.Sprintf(
				"ℹ Порт %d используется %d инбаундом(ами). HAProxy обеспечит корректную маршрутизацию через SNI.",
				port, len(result.Conflicts),
			)
			result.Action = ActionAllow
		} else if result.SharingMechanism == "path" {
			result.Severity = SeverityInfo
			result.Message = fmt.Sprintf(
				"ℹ Порт %d используется %d инбаундом(ами). HAProxy обеспечит корректную маршрутизацию через Path.",
				port, len(result.Conflicts),
			)
			result.Action = ActionAllow
		} else {
			result.Severity = SeverityWarning
			result.Message = fmt.Sprintf(
				"⚠ Порт %d уже используется инбаундом '%s' (%s/%s). HAProxy может обеспечить совместную работу, но убедитесь, что SNI/Path отличаются от существующих.",
				port,
				result.Conflicts[0].InboundName,
				result.Conflicts[0].Protocol,
				result.Conflicts[0].Transport,
			)
			result.Action = ActionConfirm
		}
	} else {
		result.Severity = SeverityError

		var reasons []string
		for _, c := range result.Conflicts {
			if !c.HaproxyCompatible {
				reasons = append(reasons, fmt.Sprintf("%s не поддерживает HAProxy", c.InboundName))
			}
		}

		if len(reasons) > 0 {
			result.Message = fmt.Sprintf(
				"✗ Порт %d уже используется и НЕ может быть совместно использован: %s. Выберите другой порт или удалите конфликтующие инбаунды.",
				port,
				strings.Join(reasons, ", "),
			)
		} else {
			result.Message = fmt.Sprintf(
				"✗ Порт %d уже используется инбаундом '%s'. Протоколы несовместимы для совместной работы через HAProxy.",
				port,
				result.Conflicts[0].InboundName,
			)
		}
		result.Action = ActionBlock
	}

	return result
}

func (v *PortValidator) isPortOverlap(port1 int, addr1 string, port2 int, addr2 string) bool {
	if port1 != port2 {
		return false
	}

	if addr1 == "0.0.0.0" || addr2 == "0.0.0.0" {
		return true
	}

	return addr1 == addr2
}

func (v *PortValidator) ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func (v *PortValidator) extractTransportFromConfig(configJSON string) string {
	if configJSON == "" {
		return ""
	}
	if idx := strings.Index(configJSON, `"transport"`); idx != -1 {
		sub := configJSON[idx:]
		if colonIdx := strings.Index(sub, `:`); colonIdx != -1 {
			valPart := sub[colonIdx+1:]
			if quoteIdx := strings.Index(valPart, `"`); quoteIdx != -1 {
				valPart = valPart[quoteIdx+1:]
				if endQuote := strings.Index(valPart, `"`); endQuote != -1 {
					return valPart[:endQuote]
				}
			}
		}
	}
	return ""
}

func (v *PortValidator) extractCoreTypeFromConfig(configJSON string, coreID uint) string {
	if configJSON != "" {
		if idx := strings.Index(configJSON, `"core_type"`); idx != -1 {
			sub := configJSON[idx:]
			if colonIdx := strings.Index(sub, `:`); colonIdx != -1 {
				valPart := sub[colonIdx+1:]
				if quoteIdx := strings.Index(valPart, `"`); quoteIdx != -1 {
					valPart = valPart[quoteIdx+1:]
					if endQuote := strings.Index(valPart, `"`); endQuote != -1 {
						return valPart[:endQuote]
					}
				}
			}
		}
	}
	switch coreID {
	case 1:
		return "xray"
	case 2:
		return "singbox"
	case 3:
		return "mihomo"
	default:
		return "xray"
	}
}
