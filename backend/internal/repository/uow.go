package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// UnitOfWork defines the interface for transactional operations
type UnitOfWork interface {
	// UserRepository returns the user repository within the transaction
	UserRepository() UserRepository

	// InboundRepository returns the inbound repository within the transaction
	InboundRepository() InboundRepository

	// OutboundRepository returns the outbound repository within the transaction
	OutboundRepository() OutboundRepository

	// CoreRepository returns the core repository within the transaction
	CoreRepository() CoreRepository

	// AdminRepository returns the admin repository within the transaction
	AdminRepository() AdminRepository

	// BackupRepository returns the backup repository within the transaction
	BackupRepository() BackupRepository

	// Commit commits the transaction
	Commit() error

	// Rollback rolls back the transaction
	Rollback() error
}

// UnitOfWorkFactory creates UnitOfWork instances
type UnitOfWorkFactory interface {
	// WithTransaction executes the given function within a transaction
	WithTransaction(ctx context.Context, fn func(uow UnitOfWork) error) error
}

// GORMUnitOfWork implements UnitOfWork using GORM
type GORMUnitOfWork struct {
	tx *gorm.DB

	userRepo     UserRepository
	inboundRepo  InboundRepository
	outboundRepo OutboundRepository
	coreRepo     CoreRepository
	adminRepo    AdminRepository
	backupRepo   BackupRepository
}

// NewGORMUnitOfWork creates a new GORM-based UnitOfWork
func NewGORMUnitOfWork(tx *gorm.DB) *GORMUnitOfWork {
	return &GORMUnitOfWork{
		tx:           tx,
		userRepo:     NewGORMUserRepository(tx),
		inboundRepo:  NewGORMInboundRepository(tx),
		outboundRepo: NewGORMOutboundRepository(tx),
		coreRepo:     NewGORMCoreRepository(tx),
		adminRepo:    NewGORMAdminRepository(tx),
		backupRepo:   NewGORMBackupRepository(tx),
	}
}

// UserRepository returns the user repository within the transaction
func (u *GORMUnitOfWork) UserRepository() UserRepository {
	return u.userRepo
}

// InboundRepository returns the inbound repository within the transaction
func (u *GORMUnitOfWork) InboundRepository() InboundRepository {
	return u.inboundRepo
}

// OutboundRepository returns the outbound repository within the transaction
func (u *GORMUnitOfWork) OutboundRepository() OutboundRepository {
	return u.outboundRepo
}

// CoreRepository returns the core repository within the transaction
func (u *GORMUnitOfWork) CoreRepository() CoreRepository {
	return u.coreRepo
}

// AdminRepository returns the admin repository within the transaction
func (u *GORMUnitOfWork) AdminRepository() AdminRepository {
	return u.adminRepo
}

// BackupRepository returns the backup repository within the transaction
func (u *GORMUnitOfWork) BackupRepository() BackupRepository {
	return u.backupRepo
}

// Commit commits the transaction
func (u *GORMUnitOfWork) Commit() error {
	return u.tx.Commit().Error
}

// Rollback rolls back the transaction
func (u *GORMUnitOfWork) Rollback() error {
	return u.tx.Rollback().Error
}

// GORMUnitOfWorkFactory implements UnitOfWorkFactory using GORM
type GORMUnitOfWorkFactory struct {
	db *gorm.DB
}

// NewGORMUnitOfWorkFactory creates a new GORM-based UnitOfWorkFactory
func NewGORMUnitOfWorkFactory(db *gorm.DB) *GORMUnitOfWorkFactory {
	return &GORMUnitOfWorkFactory{db: db}
}

// WithTransaction executes the given function within a transaction
func (f *GORMUnitOfWorkFactory) WithTransaction(ctx context.Context, fn func(uow UnitOfWork) error) error {
	tx := f.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return WrapError(tx.Error)
	}

	uow := NewGORMUnitOfWork(tx)

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(uow); err != nil {
		if rbErr := tx.Rollback().Error; rbErr != nil {
			return fmt.Errorf("%w: transaction rollback failed: %w", err, WrapError(rbErr))
		}
		return err
	}

	if err := tx.Commit().Error; err != nil {
		return WrapError(fmt.Errorf("transaction commit failed: %w", err))
	}

	return nil
}
