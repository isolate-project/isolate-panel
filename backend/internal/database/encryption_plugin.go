package database

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"gorm.io/gorm"

	"github.com/isolate-project/isolate-panel/internal/crypto"
)

// EncryptionPlugin is a GORM plugin that automatically encrypts/decrypts configured fields
type EncryptionPlugin struct {
	enc    *crypto.FieldEncrypter
	fields map[string][]string // table -> encrypted field names
	mu     sync.RWMutex
}

// NewEncryptionPlugin creates a new encryption plugin with the given encrypter
func NewEncryptionPlugin(enc *crypto.FieldEncrypter) *EncryptionPlugin {
	return &EncryptionPlugin{
		enc:    enc,
		fields: make(map[string][]string),
	}
}

// NewEncryptionPluginFromEnv creates a new encryption plugin using environment-based key loading
func NewEncryptionPluginFromEnv() (*EncryptionPlugin, error) {
	enc, err := crypto.NewFieldEncrypterFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to create field encrypter: %w", err)
	}
	return NewEncryptionPlugin(enc), nil
}

// RegisterField registers a field for automatic encryption/decryption
// tableName: the database table name (e.g., "users")
// fieldName: the struct field name (e.g., "Token")
func (ep *EncryptionPlugin) RegisterField(tableName, fieldName string) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.fields[tableName] = append(ep.fields[tableName], fieldName)
}

// RegisterFields registers multiple fields for a table
func (ep *EncryptionPlugin) RegisterFields(tableName string, fieldNames ...string) {
	ep.mu.Lock()
	defer ep.mu.Unlock()
	ep.fields[tableName] = append(ep.fields[tableName], fieldNames...)
}

// Name returns the plugin name for GORM
func (ep *EncryptionPlugin) Name() string {
	return "encryption_plugin"
}

// Initialize registers the callbacks with GORM
func (ep *EncryptionPlugin) Initialize(db *gorm.DB) error {
	// Register BeforeCreate callback
	if err := db.Callback().Create().Before("gorm:create").Register("encryption:before_create", ep.beforeCreate); err != nil {
		return fmt.Errorf("failed to register before_create callback: %w", err)
	}

	// Register BeforeUpdate callback
	if err := db.Callback().Update().Before("gorm:update").Register("encryption:before_update", ep.beforeUpdate); err != nil {
		return fmt.Errorf("failed to register before_update callback: %w", err)
	}

	// Register AfterFind callback
	if err := db.Callback().Query().After("gorm:query").Register("encryption:after_find", ep.afterFind); err != nil {
		return fmt.Errorf("failed to register after_find callback: %w", err)
	}

	return nil
}

// beforeCreate encrypts fields before creating a record
func (ep *EncryptionPlugin) beforeCreate(db *gorm.DB) {
	if db.Error != nil || db.Statement.Schema == nil {
		return
	}

	tableName := db.Statement.Schema.Table
	ep.mu.RLock()
	fields, ok := ep.fields[tableName]
	ep.mu.RUnlock()

	if !ok || len(fields) == 0 {
		return
	}

	reflectValue := reflect.ValueOf(db.Statement.Dest)
	if reflectValue.Kind() != reflect.Ptr {
		return
	}
	reflectValue = reflectValue.Elem()

	// Handle slice inserts
	if reflectValue.Kind() == reflect.Slice {
		for i := 0; i < reflectValue.Len(); i++ {
			ep.encryptFields(reflectValue.Index(i), fields)
		}
	} else {
		ep.encryptFields(reflectValue, fields)
	}
}

// beforeUpdate encrypts fields before updating a record
func (ep *EncryptionPlugin) beforeUpdate(db *gorm.DB) {
	if db.Error != nil || db.Statement.Schema == nil {
		return
	}

	tableName := db.Statement.Schema.Table
	ep.mu.RLock()
	fields, ok := ep.fields[tableName]
	ep.mu.RUnlock()

	if !ok || len(fields) == 0 {
		return
	}

	reflectValue := reflect.ValueOf(db.Statement.Dest)
	if reflectValue.Kind() != reflect.Ptr {
		return
	}
	reflectValue = reflectValue.Elem()

	// Handle slice updates
	if reflectValue.Kind() == reflect.Slice {
		for i := 0; i < reflectValue.Len(); i++ {
			ep.encryptFields(reflectValue.Index(i), fields)
		}
	} else {
		ep.encryptFields(reflectValue, fields)
	}
}

// afterFind decrypts fields after finding a record
func (ep *EncryptionPlugin) afterFind(db *gorm.DB) {
	if db.Error != nil || db.Statement.Schema == nil {
		return
	}

	tableName := db.Statement.Schema.Table
	ep.mu.RLock()
	fields, ok := ep.fields[tableName]
	ep.mu.RUnlock()

	if !ok || len(fields) == 0 {
		return
	}

	reflectValue := reflect.ValueOf(db.Statement.Dest)
	if reflectValue.Kind() != reflect.Ptr {
		return
	}
	reflectValue = reflectValue.Elem()

	// Handle slice results
	if reflectValue.Kind() == reflect.Slice {
		for i := 0; i < reflectValue.Len(); i++ {
			ep.decryptFields(reflectValue.Index(i), fields)
		}
	} else {
		ep.decryptFields(reflectValue, fields)
	}
}

// encryptFields encrypts the specified fields of a struct
func (ep *EncryptionPlugin) encryptFields(v reflect.Value, fields []string) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	for _, fieldName := range fields {
		field := v.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		// Handle string fields
		if field.Kind() == reflect.String && field.CanSet() {
			plaintext := field.String()
			if plaintext == "" {
				continue
			}

			// Check if already encrypted
			if ep.enc.IsEncrypted(plaintext) {
				continue
			}

			ciphertext, err := ep.enc.Encrypt(plaintext)
			if err != nil {
				continue // Silently skip encryption on error
			}
			field.SetString(ciphertext)
		}

		// Handle pointer to string fields
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.String {
			if field.IsNil() {
				continue
			}
			plaintext := field.Elem().String()
			if plaintext == "" {
				continue
			}

			// Check if already encrypted
			if ep.enc.IsEncrypted(plaintext) {
				continue
			}

			ciphertext, err := ep.enc.Encrypt(plaintext)
			if err != nil {
				continue
			}
			field.Elem().SetString(ciphertext)
		}
	}
}

// decryptFields decrypts the specified fields of a struct
func (ep *EncryptionPlugin) decryptFields(v reflect.Value, fields []string) {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}

	for _, fieldName := range fields {
		field := v.FieldByName(fieldName)
		if !field.IsValid() {
			continue
		}

		// Handle string fields
		if field.Kind() == reflect.String && field.CanSet() {
			ciphertext := field.String()
			if ciphertext == "" {
				continue
			}

			// Check if looks encrypted
			if !ep.enc.IsEncrypted(ciphertext) {
				continue
			}

			plaintext, err := ep.enc.Decrypt(ciphertext)
			if err != nil {
				continue // Silently skip decryption on error
			}
			field.SetString(plaintext)
		}

		// Handle pointer to string fields
		if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.String {
			if field.IsNil() {
				continue
			}
			ciphertext := field.Elem().String()
			if ciphertext == "" {
				continue
			}

			// Check if looks encrypted
			if !ep.enc.IsEncrypted(ciphertext) {
				continue
			}

			plaintext, err := ep.enc.Decrypt(ciphertext)
			if err != nil {
				continue
			}
			field.Elem().SetString(plaintext)
		}
	}
}

// DefaultEncryptedFields returns the default field encryption configuration
// for all sensitive fields across the application models
func DefaultEncryptedFields() map[string][]string {
	return map[string][]string{
		"users": {
			"Token",
			"SubscriptionToken",
		},
		"admins": {
			"TOTPSecret",
		},
		"notification_settings": {
			"WebhookSecret",
			"TelegramBotToken",
		},
		"inbounds": {
			"ConfigJSON",
			"RealityConfigJSON",
		},
	}
}

// RegisterDefaultFields registers all default encrypted fields with the plugin
func (ep *EncryptionPlugin) RegisterDefaultFields() {
	for table, fields := range DefaultEncryptedFields() {
		ep.RegisterFields(table, fields...)
	}
}

// IsFieldEncrypted checks if a field is registered for encryption
func (ep *EncryptionPlugin) IsFieldEncrypted(tableName, fieldName string) bool {
	ep.mu.RLock()
	defer ep.mu.RUnlock()

	fields, ok := ep.fields[tableName]
	if !ok {
		return false
	}

	for _, f := range fields {
		if strings.EqualFold(f, fieldName) {
			return true
		}
	}
	return false
}
