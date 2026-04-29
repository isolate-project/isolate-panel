package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/models"
)

// WebAuthnService handles FIDO2/WebAuthn registration and authentication
type WebAuthnService struct {
	db        *gorm.DB
	webAuthn  *webauthn.WebAuthn
	sessions  map[string]*webauthn.SessionData
	sessionMu sync.RWMutex
}

// webAuthnUser implements webauthn.User interface
type webAuthnUser struct {
	id          uint
	name        string
	displayName string
	credentials []webauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte {
	return []byte(fmt.Sprintf("%d", u.id))
}

func (u *webAuthnUser) WebAuthnName() string {
	return u.name
}

func (u *webAuthnUser) WebAuthnDisplayName() string {
	return u.displayName
}

func (u *webAuthnUser) WebAuthnIcon() string {
	return ""
}

func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// NewWebAuthnService creates a new WebAuthn service
func NewWebAuthnService(db *gorm.DB, rpID, rpOrigin, rpName string) (*WebAuthnService, error) {
	wconfig := &webauthn.Config{
		RPDisplayName: rpName,
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},
		AttestationPreference: protocol.PreferNoAttestation,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			RequireResidentKey:      protocol.ResidentKeyNotRequired(),
			UserVerification:        protocol.VerificationPreferred,
		},
		Timeouts: webauthn.TimeoutsConfig{
			Login: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    120000000000,
				TimeoutUVD: 120000000000,
			},
			Registration: webauthn.TimeoutConfig{
				Enforce:    true,
				Timeout:    120000000000,
				TimeoutUVD: 120000000000,
			},
		},
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create webauthn instance: %w", err)
	}

	return &WebAuthnService{
		db:       db,
		webAuthn: w,
		sessions: make(map[string]*webauthn.SessionData),
	}, nil
}

// BeginRegistration starts the registration process for a new WebAuthn credential
func (s *WebAuthnService) BeginRegistration(adminID uint) (*protocol.CredentialCreation, error) {
	var admin models.Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return nil, fmt.Errorf("admin not found: %w", err)
	}

	user := &webAuthnUser{
		id:          admin.ID,
		name:        admin.Username,
		displayName: admin.Username,
	}

	existingCreds, err := s.loadCredentialsForUser(adminID)
	if err != nil {
		return nil, fmt.Errorf("failed to load existing credentials: %w", err)
	}
	user.credentials = existingCreds

	options, sessionData, err := s.webAuthn.BeginRegistration(user)
	if err != nil {
		return nil, fmt.Errorf("failed to begin registration: %w", err)
	}

	s.storeSession(sessionData.Challenge, sessionData)

	return options, nil
}

// FinishRegistration completes the registration process and stores the credential
func (s *WebAuthnService) FinishRegistration(adminID uint, response *protocol.ParsedCredentialCreationData) error {
	var admin models.Admin
	if err := s.db.First(&admin, adminID).Error; err != nil {
		return fmt.Errorf("admin not found: %w", err)
	}

	user := &webAuthnUser{
		id:          admin.ID,
		name:        admin.Username,
		displayName: admin.Username,
	}

	// Get the challenge from the response to lookup session
	challenge := response.Response.CollectedClientData.Challenge
	if challenge == "" {
		return fmt.Errorf("challenge not found in client data")
	}

	session := s.getAndDeleteSession(challenge)
	if session == nil {
		return fmt.Errorf("registration session expired or not found")
	}

	credential, err := s.webAuthn.CreateCredential(user, *session, response)
	if err != nil {
		return fmt.Errorf("failed to create credential: %w", err)
	}

	transportJSON, _ := json.Marshal(response.Response.Transports)

	cred := models.WebAuthnCredential{
		AdminID:          adminID,
		CredentialID:     base64.URLEncoding.EncodeToString(credential.ID),
		PublicKey:        credential.PublicKey,
		AttestationType:  string(response.Response.AttestationObject.Type),
		Transport:        string(transportJSON),
		SignCount:        credential.Authenticator.SignCount,
		AAGUID:           base64.URLEncoding.EncodeToString(response.Response.AttestationObject.AuthData.AttData.AAGUID),
		IsBackupEligible: response.Response.AttestationObject.AuthData.Flags.HasBackupEligible(),
		IsBackup:         response.Response.AttestationObject.AuthData.Flags.HasBackupState(),
		Attachment:       string(protocol.Platform),
	}

	if err := s.db.Create(&cred).Error; err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	return nil
}

// BeginAuthentication starts the authentication process
func (s *WebAuthnService) BeginAuthentication() (*protocol.CredentialAssertion, error) {
	options, sessionData, err := s.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin authentication: %w", err)
	}

	s.storeSession(sessionData.Challenge, sessionData)

	return options, nil
}

// FinishAuthentication completes the authentication process
func (s *WebAuthnService) FinishAuthentication(response *protocol.ParsedCredentialAssertionData) (uint, error) {
	// Get the challenge from the response to lookup session
	challenge := response.Response.CollectedClientData.Challenge
	if challenge == "" {
		return 0, fmt.Errorf("challenge not found in client data")
	}

	session := s.getAndDeleteSession(challenge)
	if session == nil {
		return 0, fmt.Errorf("authentication session expired or not found")
	}

	// Find the credential by ID
	credentialID := base64.URLEncoding.EncodeToString(response.RawID)

	var storedCred models.WebAuthnCredential
	if err := s.db.Where("credential_id = ?", credentialID).First(&storedCred).Error; err != nil {
		return 0, fmt.Errorf("credential not found: %w", err)
	}

	var admin models.Admin
	if err := s.db.First(&admin, storedCred.AdminID).Error; err != nil {
		return 0, fmt.Errorf("admin not found: %w", err)
	}

	if !admin.IsActive {
		return 0, fmt.Errorf("admin account is deactivated")
	}

	user := &webAuthnUser{
		id:          admin.ID,
		name:        admin.Username,
		displayName: admin.Username,
	}

	cred := webauthn.Credential{
		ID:              response.RawID,
		PublicKey:       storedCred.PublicKey,
		AttestationType: storedCred.AttestationType,
		Transport:       parseTransports(storedCred.Transport),
		Authenticator: webauthn.Authenticator{
			AAGUID:    []byte(storedCred.AAGUID),
			SignCount: storedCred.SignCount,
		},
	}
	user.credentials = []webauthn.Credential{cred}

	validatedCred, err := s.webAuthn.ValidateLogin(user, *session, response)
	if err != nil {
		return 0, fmt.Errorf("authentication validation failed: %w", err)
	}

	now := time.Now()
	storedCred.LastUsedAt = &now
	storedCred.SignCount = validatedCred.Authenticator.SignCount
	s.db.Save(&storedCred)

	return storedCred.AdminID, nil
}

// GetCredentialsForAdmin returns all WebAuthn credentials for an admin
func (s *WebAuthnService) GetCredentialsForAdmin(adminID uint) ([]models.WebAuthnCredential, error) {
	var creds []models.WebAuthnCredential
	if err := s.db.Where("admin_id = ?", adminID).Find(&creds).Error; err != nil {
		return nil, err
	}
	return creds, nil
}

// DeleteCredential removes a WebAuthn credential
func (s *WebAuthnService) DeleteCredential(adminID uint, credentialID string) error {
	result := s.db.Where("admin_id = ? AND credential_id = ?", adminID, credentialID).Delete(&models.WebAuthnCredential{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("credential not found")
	}
	return nil
}

// HasCredentials checks if an admin has any WebAuthn credentials
func (s *WebAuthnService) HasCredentials(adminID uint) bool {
	var count int64
	s.db.Model(&models.WebAuthnCredential{}).Where("admin_id = ?", adminID).Count(&count)
	return count > 0
}

// loadCredentialsForUser loads existing credentials for an admin
func (s *WebAuthnService) loadCredentialsForUser(adminID uint) ([]webauthn.Credential, error) {
	var storedCreds []models.WebAuthnCredential
	if err := s.db.Where("admin_id = ?", adminID).Find(&storedCreds).Error; err != nil {
		return nil, err
	}

	var creds []webauthn.Credential
	for _, sc := range storedCreds {
		credID, _ := base64.URLEncoding.DecodeString(sc.CredentialID)
		creds = append(creds, webauthn.Credential{
			ID:              credID,
			PublicKey:       sc.PublicKey,
			AttestationType: sc.AttestationType,
			Transport:       parseTransports(sc.Transport),
			Authenticator: webauthn.Authenticator{
				AAGUID:    []byte(sc.AAGUID),
				SignCount: sc.SignCount,
			},
		})
	}
	return creds, nil
}

// storeSession stores session data in memory
func (s *WebAuthnService) storeSession(challenge string, data *webauthn.SessionData) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.sessions[challenge] = data
}

// getAndDeleteSession retrieves and removes session data
func (s *WebAuthnService) getAndDeleteSession(challenge string) *webauthn.SessionData {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	data, ok := s.sessions[challenge]
	if !ok {
		return nil
	}
	delete(s.sessions, challenge)
	return data
}

// CleanupExpiredSessions removes expired sessions (should be called periodically)
func (s *WebAuthnService) CleanupExpiredSessions() {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	now := time.Now()
	for id, session := range s.sessions {
		if now.After(session.Expires) {
			delete(s.sessions, id)
		}
	}
}

// parseTransports parses JSON transport array
func parseTransports(transportJSON string) []protocol.AuthenticatorTransport {
	var transports []string
	if err := json.Unmarshal([]byte(transportJSON), &transports); err != nil {
		return nil
	}

	var result []protocol.AuthenticatorTransport
	for _, t := range transports {
		result = append(result, protocol.AuthenticatorTransport(t))
	}
	return result
}
