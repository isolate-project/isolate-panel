# Smart Warning UI Validation System

## Overview

The Smart Warning UI Validation System provides real-time port conflict detection with intelligent severity classification. It helps administrators understand whether multiple inbounds can share the same port through HAProxy routing (SNI-based or Path-based) or if conflicts require selecting a different port.

---

## Severity Levels Table

| Severity | Icon | Color (Hex) | Russian Message | Action |
|----------|------|-------------|-----------------|--------|
| **INFO** | ✓ | 🟢 `#52c41a` (Green) | "Порт свободен" / "HAProxy обеспечит маршрутизацию" | ✅ **Allow** - Proceed with creation |
| **WARNING** | ⚠️ | 🟡 `#faad14` (Yellow) | "Порт используется, но HAProxy поддерживает совместную работу через SNI/Path" | ⚠️ **Confirm** - Show confirmation dialog |
| **ERROR** | ✗ | 🔴 `#f5222d` (Red) | "Протоколы несовместимы. Выберите другой порт." | ❌ **Block** - Prevent creation |

### Severity Determination Logic

```
IF port has NO conflicts:
    → Severity: INFO (Green)
    → Message: "✓ Порт свободен"
    → Action: Allow

ELSE IF port has conflicts AND can share via HAProxy:
    → Severity: WARNING (Yellow)
    → Message: "⚠ Порт {port} уже используется. HAProxy может обеспечить совместную работу через {sni|path}-based routing."
    → Action: Confirm (requires user confirmation)

ELSE (port has conflicts AND cannot share):
    → Severity: ERROR (Red)
    → Message: "✗ Порт {port} уже используется. Протоколы несовместимы для совместной работы через HAProxy."
    → Action: Block
```

---

## Backend Logic

### PortConflictCheck Struct

**File**: `backend/internal/haproxy/validation.go` (lines 197-207)

```go
type PortConflictCheck struct {
    Port              int                `json:"port"`
    IsAvailable       bool               `json:"is_available"`
    HaproxyCompatible bool               `json:"haproxy_compatible"`
    CanSharePort      bool               `json:"can_share_port"`
    SharingMechanism  string             `json:"sharing_mechanism"` // "sni", "path"
    Severity          ValidationSeverity `json:"severity"`            // "info", "warning", "error"
    Action            ValidationAction   `json:"action"`            // "allow", "confirm", "block"
    Message           string             `json:"message"`             // Russian
    Conflicts         []PortConflict     `json:"conflicts,omitempty"`
}
```

### PortConflict Struct

```go
type PortConflict struct {
    InboundID       uint   `json:"inbound_id"`
    InboundName     string `json:"inbound_name"`
    Protocol        string `json:"protocol"`
    Transport       string `json:"transport"`
    Port            int    `json:"port"`
    Listen          string `json:"listen"`
    CanShare        bool   `json:"can_share"`
    SameProtocol    bool   `json:"same_protocol"`
    RequiresConfirm bool   `json:"requires_confirm"`
}
```

### Severity and Action Enums

```go
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
```

---

## PortValidator Algorithm

### Version 1: Basic Algorithm (HAPROXY_IMPLEMENTATION_PLAN lines 209-270)

**File**: `backend/internal/haproxy/validation.go`

```go
func (v *PortValidator) ValidatePortConflict(
    port int, listen, protocol, transport, coreType string,
    existingInbounds []Inbound,
) *PortConflictCheck {
    result := &PortConflictCheck{Port: port}
    
    // Step 1: Check HAProxy compatibility
    result.HaproxyCompatible = v.isHaproxyCompatible(protocol, transport, coreType)
    
    // Step 2: Find conflicts
    for _, existing := range existingInbounds {
        if !v.isPortOverlap(port, listen, existing.Port, existing.Listen) {
            continue
        }
        
        conflict := PortConflict{
            InboundID:   existing.ID,
            InboundName: existing.Remark,
            Protocol:    existing.Protocol,
            Transport:   existing.Transport,
        }
        
        // Step 3: Can they share via HAProxy?
        existingCompatible := v.isHaproxyCompatible(existing.Protocol, existing.Transport, existing.CoreType)
        if result.HaproxyCompatible && existingCompatible {
            conflict.CanShare = true
            result.CanSharePort = true
            
            // Step 4: Determine sharing mechanism
            if v.supportsSNI(protocol) && v.supportsSNI(existing.Protocol) {
                result.SharingMechanism = "sni"
            } else if v.supportsPath(transport) && v.supportsPath(existing.Transport) {
                result.SharingMechanism = "path"
            }
        }
        
        result.Conflicts = append(result.Conflicts, conflict)
    }
    
    // Step 5-7: Determine severity and message (Russian)
    if len(result.Conflicts) == 0 {
        // No conflicts - Port is free
        result.Severity = SeverityInfo
        result.Message = "✓ Порт свободен"
        result.Action = ActionAllow
    } else if result.CanSharePort {
        // Can share via HAProxy - Warning
        result.Severity = SeverityWarning
        result.Message = fmt.Sprintf(
            "⚠ Порт %d уже используется. HAProxy может обеспечить совместную работу через %s-based routing.",
            port, result.SharingMechanism,
        )
        result.Action = ActionConfirm
    } else {
        // Cannot share - Error
        result.Severity = SeverityError
        result.Message = fmt.Sprintf(
            "✗ Порт %d уже используется. Протоколы несовместимы для совместной работы через HAProxy.",
            port,
        )
        result.Action = ActionBlock
    }
    
    return result
}
```

### Version 2: Enhanced Algorithm (HAPROXY_MULTI_PORT_IMPLEMENTATION lines 589-728)

**File**: `backend/internal/haproxy/validation.go`

```go
// ValidatePortConflict checks if a port can be used for a new inbound
func (v *PortValidator) ValidatePortConflict(
    port int,
    listen string,
    protocol string,
    transport string,
    coreType string,
    existingInbounds []Inbound,
) *PortConflictCheck {
    
    result := &PortConflictCheck{
        Port:      port,
        Listen:    listen,
        Protocol:  protocol,
        Transport: transport,
    }
    
    // Step 1: Check HAProxy compatibility for new inbound
    result.HaproxyCompatible = v.isHaproxyCompatible(protocol, transport, coreType)
    
    // Step 2: Find all conflicting inbounds
    for _, existing := range existingInbounds {
        if !v.isPortOverlap(port, listen, existing.Port, existing.Listen) {
            continue
        }
        
        conflict := PortConflict{
            InboundID:   existing.ID,
            InboundName: existing.Remark,
            Protocol:    existing.Protocol,
            Transport:   existing.Transport,
            Port:        existing.Port,
            Listen:      existing.Listen,
        }
        
        // Step 3: Check if protocols can share the port
        conflict.SameProtocol = (protocol == existing.Protocol)
        
        // Step 4: Check HAProxy compatibility for existing inbound
        existingCompatible := v.isHaproxyCompatible(
            existing.Protocol, 
            existing.Transport, 
            existing.CoreType,
        )
        
        // Step 5: Can they share via HAProxy?
        if result.HaproxyCompatible && existingCompatible {
            // Both support HAProxy - can share via SNI or Path
            conflict.CanShare = true
            result.CanSharePort = true
            
            // Determine sharing mechanism
            if v.supportsSNI(protocol) && v.supportsSNI(existing.Protocol) {
                result.SharingMechanism = "sni"
            } else if v.supportsPath(transport) && v.supportsPath(existing.Transport) {
                result.SharingMechanism = "path"
            }
        } else {
            // At least one doesn't support HAProxy
            conflict.CanShare = false
        }
        
        // Step 6: Check if confirmation is needed
        if conflict.CanShare && !conflict.SameProtocol {
            // Different protocols sharing via HAProxy - should confirm
            conflict.RequiresConfirm = true
        }
        
        result.Conflicts = append(result.Conflicts, conflict)
    }
    
    // Step 7: Determine severity and action
    result.IsAvailable = len(result.Conflicts) == 0 || result.CanSharePort
    
    if len(result.Conflicts) == 0 {
        // Port is completely free
        result.Severity = SeverityInfo
        result.Message = "✓ Порт свободен"
        result.Action = ActionAllow
        
    } else if result.CanSharePort {
        // Port in use but can share via HAProxy
        if len(result.Conflicts) == 1 && result.Conflicts[0].RequiresConfirm {
            // Single conflict, different protocols - needs confirmation
            result.Severity = SeverityWarning
            result.Message = fmt.Sprintf(
                "⚠ Порт %d уже используется инбаундом '%s' (%s/%s). " +
                "HAProxy может обеспечить совместную работу через %s-based routing. " +
                "Убедитесь, что SNI/Path отличаются.",
                port,
                result.Conflicts[0].InboundName,
                result.Conflicts[0].Protocol,
                result.Conflicts[0].Transport,
                result.SharingMechanism,
            )
            result.Action = ActionConfirm
            
        } else {
            // Multiple conflicts or same protocol - info level
            result.Severity = SeverityInfo
            result.Message = fmt.Sprintf(
                "ℹ Порт %d используется %d инбаундом(ами). " +
                "HAProxy обеспечит корректную маршрутизацию.",
                port, len(result.Conflicts),
            )
            result.Action = ActionAllow
        }
        
    } else {
        // Port in use and cannot share
        result.Severity = SeverityError
        
        // Find the specific reason
        nonCompatibleConflicts := []string{}
        for _, c := range result.Conflicts {
            if !v.isHaproxyCompatible(c.Protocol, c.Transport, "") {
                nonCompatibleConflicts = append(nonCompatibleConflicts, c.InboundName)
            }
        }
        
        if len(nonCompatibleConflicts) > 0 {
            result.Message = fmt.Sprintf(
                "✗ Порт %d уже используется и НЕ может быть совместно использован: " +
                "следующие инбаунды не поддерживают HAProxy: %s. " +
                "Выберите другой порт или удалите конфликтующие инбаунды.",
                port,
                strings.Join(nonCompatibleConflicts, ", "),
            )
        } else {
            result.Message = fmt.Sprintf(
                "✗ Порт %d уже используется инбаундом '%s'. " +
                "Протоколы несовместимы для совместной работы через HAProxy.",
                port,
                result.Conflicts[0].InboundName,
            )
        }
        result.Action = ActionBlock
    }
    
    return result
}
```

### Helper Functions

```go
// isHaproxyCompatible checks if a protocol/transport/core combination works with HAProxy
func (v *PortValidator) isHaproxyCompatible(protocol, transport, coreType string) bool {
    // UDP-based transports never compatible
    if isUDPTransport(transport) {
        return false
    }
    
    // Check core-specific limitations
    switch coreType {
    case "xray":
        // Xray gRPC doesn't support PROXY protocol (but can still route)
        // Xray XHTTP doesn't support PROXY v2 (uses X-Forwarded-For)
        return true // All TCP-based work, just without PROXY v2 for some
        
    case "singbox":
        // Sing-box: all TCP-based work but NO PROXY protocol v2 at all
        // (removed in 1.6.0+)
        return true // Can route, but no real IP preservation
        
    case "mihomo":
        // Mihomo: no PROXY protocol, but all TCP-based work
        return true // Can route, but no real IP preservation
        
    default:
        return false
    }
}

// supportsSNI checks if protocol supports SNI-based routing
func (v *PortValidator) supportsSNI(protocol string) bool {
    // All TLS-based protocols support SNI
    tlsProtocols := []string{"vless", "vmess", "trojan", "shadowtls", "anytls"}
    return contains(tlsProtocols, protocol)
}

// supportsPath checks if transport supports path-based routing
func (v *PortValidator) supportsPath(transport string) bool {
    // HTTP-based transports support path routing
    pathTransports := []string{"ws", "websocket", "httpupgrade", "xhttp", "grpc"}
    return contains(pathTransports, transport)
}

// isUDPTransport checks if transport is UDP-based
func isUDPTransport(transport string) bool {
    udpTransports := []string{"quic", "kcp", "hysteria", "hysteria2", "tuic"}
    return contains(udpTransports, transport)
}
```

---

## API Endpoint

### CheckPortAvailability Handler

**File**: `backend/internal/api/handlers/port_validation.go` (lines 951-997)

```go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "github.com/isolate-project/isolate-panel/backend/internal/haproxy"
)

// CheckPortAvailability validates if a port can be used for a new inbound
func (h *Handler) CheckPortAvailability(c *fiber.Ctx) error {
    var req struct {
        Port      int    `json:"port"`
        Listen    string `json:"listen"`
        Protocol  string `json:"protocol"`
        Transport string `json:"transport"`
        CoreType  string `json:"coreType"`
    }
    
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid request body",
        })
    }
    
    // Fetch all active inbounds
    var inbounds []haproxy.Inbound
    if err := h.db.Where("is_active = ?", true).Find(&inbounds).Error; err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": "Failed to fetch inbounds",
        })
    }
    
    // Run validation
    validator := haproxy.NewPortValidator()
    result := validator.ValidatePortConflict(
        req.Port,
        req.Listen,
        req.Protocol,
        req.Transport,
        req.CoreType,
        inbounds,
    )
    
    return c.JSON(result)
}
```

### API Specification

| Property | Value |
|----------|-------|
| **Endpoint** | `POST /api/inbounds/check-port` |
| **Content-Type** | `application/json` |
| **Auth Required** | Yes (JWT) |
| **Rate Limit** | 60 req/min |

#### Request Body

```json
{
    "port": 443,
    "listen": "0.0.0.0",
    "protocol": "vless",
    "transport": "tcp",
    "coreType": "xray"
}
```

#### Response Body (Success - INFO)

```json
{
    "port": 443,
    "is_available": true,
    "haproxy_compatible": true,
    "can_share_port": false,
    "sharing_mechanism": "",
    "severity": "info",
    "action": "allow",
    "message": "✓ Порт свободен",
    "conflicts": []
}
```

#### Response Body (Warning)

```json
{
    "port": 443,
    "is_available": true,
    "haproxy_compatible": true,
    "can_share_port": true,
    "sharing_mechanism": "sni",
    "severity": "warning",
    "action": "confirm",
    "message": "⚠ Порт 443 уже используется инбаундом 'Existing Inbound' (vmess/websocket). HAProxy может обеспечить совместную работу через sni-based routing. Убедитесь, что SNI/Path отличаются.",
    "conflicts": [
        {
            "inbound_id": 1,
            "inbound_name": "Existing Inbound",
            "protocol": "vmess",
            "transport": "websocket",
            "port": 443,
            "listen": "0.0.0.0",
            "can_share": true,
            "same_protocol": false,
            "requires_confirm": true
        }
    ]
}
```

#### Response Body (Error)

```json
{
    "port": 443,
    "is_available": false,
    "haproxy_compatible": true,
    "can_share_port": false,
    "sharing_mechanism": "",
    "severity": "error",
    "action": "block",
    "message": "✗ Порт 443 уже используется инбаундом 'UDP Inbound'. Протоколы несовместимы для совместной работы через HAProxy.",
    "conflicts": [
        {
            "inbound_id": 2,
            "inbound_name": "UDP Inbound",
            "protocol": "vless",
            "transport": "quic",
            "port": 443,
            "listen": "0.0.0.0",
            "can_share": false,
            "same_protocol": false,
            "requires_confirm": false
        }
    ]
}
```

---

## Frontend Components

### Version 1: Basic PortValidationField (HAPROXY_IMPLEMENTATION_PLAN lines 275-320)

**File**: `frontend/src/components/inbound/PortValidationField.tsx`

```tsx
// File: frontend/src/components/inbound/PortValidationField.tsx

export function PortValidationField({ value, onChange, protocol, transport, coreType }: Props) {
    const [state, setState] = useState<ValidationState>({ status: 'idle', message: '', action: 'allow' });
    
    // Debounced validation (500ms)
    const validatePort = useCallback(
        debounce(async (port: number) => {
            setState(prev => ({ ...prev, status: 'checking' }));
            
            const result = await fetch('/api/inbounds/check-port', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ port, protocol, transport, coreType }),
            }).then(r => r.json());
            
            setState({
                status: result.severity === 'error' ? 'error' : 
                       result.severity === 'warning' ? 'warning' : 'success',
                message: result.message,
                action: result.action,
            });
        }, 500),
        [protocol, transport, coreType]
    );
    
    return (
        <div class="port-validation-field">
            <input 
                type="number" 
                value={value} 
                onChange={e => { onChange(e.target.value); validatePort(e.target.value); }}
                style={{ borderColor: getStatusColor(state.status) }}
            />
            {state.status === 'checking' && <span>⏳ Проверка...</span>}
            {state.status === 'success' && <span style={{ color: '#52c41a' }}>✓ {state.message}</span>}
            {state.status === 'warning' && (
                <div style={{ color: '#faad14' }}>
                    ⚠️ {state.message}
                    <button onClick={() => showConfirmModal(state)}>Подтвердить</button>
                </div>
            )}
            {state.status === 'error' && <span style={{ color: '#f5222d' }}>✗ {state.message}</span>}
        </div>
    );
}
```

### Version 2: Enhanced PortValidationField (HAPROXY_MULTI_PORT_IMPLEMENTATION lines 783-946)

**File**: `frontend/src/components/inbound/PortValidationField.tsx`

```tsx
import { h } from 'preact';
import { useState, useCallback } from 'preact/hooks';
import { debounce } from 'lodash-es';

interface PortValidationProps {
    value: number;
    onChange: (port: number) => void;
    protocol: string;
    transport: string;
    coreType: string;
    listen: string;
    existingInbounds: Inbound[];
}

type ValidationState = {
    status: 'idle' | 'checking' | 'success' | 'warning' | 'error';
    message: string;
    action: 'allow' | 'confirm' | 'block';
    canShare: boolean;
    conflicts?: ConflictInfo[];
};

export function PortValidationField({
    value,
    onChange,
    protocol,
    transport,
    coreType,
    listen,
    existingInbounds,
}: PortValidationProps) {
    const [state, setState] = useState<ValidationState>({ 
        status: 'idle', 
        message: '', 
        action: 'allow',
        canShare: false,
    });
    
    // Debounced validation
    const validatePort = useCallback(
        debounce(async (port: number) => {
            if (!port || port < 1 || port > 65535) {
                setState({
                    status: 'error',
                    message: 'Порт должен быть от 1 до 65535',
                    action: 'block',
                    canShare: false,
                });
                return;
            }
            
            setState(prev => ({ ...prev, status: 'checking' }));
            
            try {
                const result = await fetch('/api/inbounds/check-port', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        port,
                        listen,
                        protocol,
                        transport,
                        coreType,
                    }),
                }).then(r => r.json());
                
                setState({
                    status: result.severity === 'error' ? 'error' : 
                           result.severity === 'warning' ? 'warning' : 'success',
                    message: result.message,
                    action: result.action,
                    canShare: result.canSharePort,
                    conflicts: result.conflicts,
                });
            } catch (e) {
                setState({
                    status: 'error',
                    message: 'Ошибка проверки порта',
                    action: 'block',
                    canShare: false,
                });
            }
        }, 500),
        [protocol, transport, coreType, listen]
    );
    
    const handleChange = (newPort: number) => {
        onChange(newPort);
        validatePort(newPort);
    };
    
    const getStatusColor = () => {
        switch (state.status) {
            case 'success': return '#52c41a';
            case 'warning': return '#faad14';
            case 'error': return '#f5222d';
            case 'checking': return '#1890ff';
            default: return undefined;
        }
    };
    
    const getIcon = () => {
        switch (state.status) {
            case 'success': return '✓';
            case 'warning': return '⚠️';
            case 'error': return '✗';
            case 'checking': return '⏳';
            default: return '';
        }
    };
    
    return (
        <div class="port-validation-field">
            <label>Порт</label>
            <div class="input-wrapper">
                <input
                    type="number"
                    value={value}
                    onInput={e => handleChange(Number(e.currentTarget.value))}
                    min={1}
                    max={65535}
                    style={{ borderColor: getStatusColor() }}
                />
                {state.status === 'checking' && <span class="spinner" />}
            </div>
            
            {state.message && (
                <div 
                    class={`validation-message ${state.status}`}
                    style={{ color: getStatusColor() }}
                >
                    <span class="icon">{getIcon()}</span>
                    {state.message}
                    
                    {state.canShare && state.conflicts && (
                        <div class="conflict-details">
                            <small>Конфликты:</small>
                            <ul>
                                {state.conflicts.map(c => (
                                    <li key={c.inboundId}>
                                        {c.inboundName} ({c.protocol}/{c.transport})
                                        {c.canShare && <span class="badge">HAProxy OK</span>}
                                    </li>
                                ))}
                            </ul>
                        </div>
                    )}
                </div>
            )}
            
            {state.status === 'warning' && state.action === 'confirm' && (
                <div class="confirm-actions">
                    <button class="btn-secondary" onClick={() => onChange(0)}>
                        Выбрать другой порт
                    </button>
                    <button class="btn-primary" onClick={() => {/* proceed with creation */}}>
                        Создать с HAProxy
                    </button>
                </div>
            )}
        </div>
    );
}
```

---

## Validation Flow Description

### Complete Validation Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         VALIDATION FLOW                                     │
└─────────────────────────────────────────────────────────────────────────────┘

    ┌──────────────┐
    │ User types   │
    │ port number  │
    └──────┬───────┘
           │
           ▼
    ┌──────────────┐     ┌─────────────────┐
    │ 500ms        │────▶│ API call        │
    │ debounce     │     │ POST /api/      │
    └──────────────┘     │ inbounds/       │
                         │ check-port      │
                         └────────┬────────┘
                                  │
                                  ▼
                         ┌─────────────────┐
                         │ Backend         │
                         │ validation      │
                         │ algorithm       │
                         └────────┬────────┘
                                  │
                    ┌─────────────┼─────────────┐
                    │             │             │
                    ▼             ▼             ▼
            ┌──────────┐  ┌──────────┐  ┌──────────┐
            │ No       │  │ Can      │  │ Cannot   │
            │ conflicts│  │ share    │  │ share    │
            └────┬─────┘  └────┬─────┘  └────┬─────┘
                 │             │             │
                 ▼             ▼             ▼
            ┌──────────┐  ┌──────────┐  ┌──────────┐
            │ INFO     │  │ WARNING  │  │ ERROR    │
            │ Green    │  │ Yellow   │  │ Red      │
            │ ✓        │  │ ⚠️       │  │ ✗        │
            └────┬─────┘  └────┬─────┘  └────┬─────┘
                 │             │             │
                 ▼             ▼             ▼
            ┌──────────┐  ┌──────────┐  ┌──────────┐
            │ Allow    │  │ Confirm  │  │ Block    │
            │ proceed  │  │ dialog   │  │ creation │
            └──────────┘  └──────────┘  └──────────┘
```

### Step-by-Step Flow

1. **User Input**: User types a port number in the input field
2. **Debouncing**: 500ms debounce prevents excessive API calls
3. **API Request**: Frontend sends `POST /api/inbounds/check-port` with:
   - `port`: The port number being validated
   - `listen`: Bind address (e.g., "0.0.0.0")
   - `protocol`: Inbound protocol (vless, vmess, trojan, etc.)
   - `transport`: Transport type (tcp, ws, grpc, quic, etc.)
   - `coreType`: Core type (xray, singbox, mihomo)

4. **Backend Processing**:
   - Fetch all active inbounds from database
   - Check HAProxy compatibility for new inbound
   - Find port conflicts with existing inbounds
   - Determine if conflicts can share via HAProxy
   - Identify sharing mechanism (SNI or Path)
   - Calculate severity level (Info/Warning/Error)
   - Generate Russian message

5. **Response Processing**:
   - Frontend receives `PortConflictCheck` JSON
   - Maps severity to UI status (success/warning/error)
   - Updates input border color based on status
   - Displays message with appropriate icon
   - Shows conflict details if available
   - Renders confirmation buttons for WARNING state

6. **User Action**:
   - **INFO**: User can proceed with inbound creation
   - **WARNING**: User must confirm to proceed or choose different port
   - **ERROR**: User must select a different port

---

## Russian Messages Reference

All Russian messages are preserved exactly as in the source code:

| Severity | Message (Russian) |
|----------|-------------------|
| INFO (Free) | `✓ Порт свободен` |
| INFO (Shared) | `ℹ Порт {port} используется {count} инбаундом(ами). HAProxy обеспечит корректную маршрутизацию.` |
| WARNING | `⚠ Порт {port} уже используется инбаундом '{name}' ({protocol}/{transport}). HAProxy может обеспечить совместную работу через {mechanism}-based routing. Убедитесь, что SNI/Path отличаются.` |
| ERROR (Incompatible) | `✗ Порт {port} уже используется инбаундом '{name}'. Протоколы несовместимы для совместной работы через HAProxy.` |
| ERROR (Non-HAProxy) | `✗ Порт {port} уже используется и НЕ может быть совместно использован: следующие инбаунды не поддерживают HAProxy: {names}. Выберите другой порт или удалите конфликтующие инбаунды.` |
| Validation Error | `Порт должен быть от 1 до 65535` |
| API Error | `Ошибка проверки порта` |
| Checking | `⏳ Проверка...` |
| Button: Change Port | `Выбрать другой порт` |
| Button: Create with HAProxy | `Создать с HAProxy` |
| Label: Conflicts | `Конфликты:` |
| Badge: HAProxy OK | `HAProxy OK` |

---

## UI States Summary

| State | Border Color | Icon | Message Color | Actions |
|-------|--------------|------|---------------|---------|
| `idle` | Default | None | None | None |
| `checking` | Blue `#1890ff` | ⏳ | Blue | Spinner shown |
| `success` | Green `#52c41a` | ✓ | Green | Allow proceed |
| `warning` | Yellow `#faad14` | ⚠️ | Yellow | Show confirm buttons |
| `error` | Red `#f5222d` | ✗ | Red | Block, show error |

---

## Key Implementation Details

### Debouncing
- **Delay**: 500ms
- **Purpose**: Prevent excessive API calls while typing
- **Library**: `lodash-es` debounce

### HAProxy Compatibility Matrix

| Core | TCP | WebSocket | gRPC | XHTTP | QUIC/Hysteria/TUIC |
|------|-----|-----------|------|-------|-------------------|
| Xray | ✅ | ✅ | ✅* | ✅** | ❌ |
| Sing-box | ✅ | ✅ | ✅ | ✅ | ❌ |
| Mihomo | ✅ | ✅ | ✅ | ✅ | ❌ |

*Xray gRPC: No PROXY v2 support (but routable)
**Xray XHTTP: Uses X-Forwarded-For instead of PROXY v2

### SNI Support
Protocols: `vless`, `vmess`, `trojan`, `shadowtls`, `anytls`

### Path Support
Transports: `ws`, `websocket`, `httpupgrade`, `xhttp`, `grpc`

### UDP Transports (No HAProxy)
`quic`, `kcp`, `hysteria`, `hysteria2`, `tuic`
