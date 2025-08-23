package repositories

import (
	"fmt"
	"strings"
	"time"

	db_models "github.com/nivschuman/VotingBlockchain/internal/database/models"
	mapping "github.com/nivschuman/VotingBlockchain/internal/mapping"
	networking_models "github.com/nivschuman/VotingBlockchain/internal/networking/models"
	"gorm.io/gorm"
)

type AddressRepository interface {
	AddressExists(address *networking_models.Address) (bool, error)
	InsertIfNotExists(address *networking_models.Address) error
	UpdateLastSeen(address *networking_models.Address, lastSeen *time.Time) error
	UpdateLastFailed(address *networking_models.Address, lastFailed *time.Time) error
	GetAddresses(limit int, excludedAddresses []*networking_models.Address) ([]*networking_models.Address, error)
	GetAddressesPaged(offset int, pageSize int, excludedAddresses []*networking_models.Address) ([]*db_models.AddressDB, int64, error)
}

type AddressRepositoryImpl struct {
	db *gorm.DB
}

func NewAddressRepositoryImpl(db *gorm.DB) *AddressRepositoryImpl {
	return &AddressRepositoryImpl{db: db}
}

func (repo *AddressRepositoryImpl) AddressExists(address *networking_models.Address) (bool, error) {
	var count int64
	result := repo.db.Model(&db_models.AddressDB{}).
		Where("ip = ? AND port = ?", address.Ip.String(), address.Port).
		Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}

func (repo *AddressRepositoryImpl) InsertIfNotExists(address *networking_models.Address) error {
	existingAddress := &db_models.AddressDB{}
	result := repo.db.Where("ip = ? AND port = ?", address.Ip.String(), address.Port).Find(existingAddress)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		addressDB := mapping.AddressToAddressDB(address)
		return repo.db.Create(addressDB).Error
	}

	return nil
}

func (repo *AddressRepositoryImpl) UpdateLastSeen(address *networking_models.Address, lastSeen *time.Time) error {
	return repo.db.Model(&db_models.AddressDB{}).
		Where("ip = ? AND port = ?", address.Ip.String(), address.Port).
		Update("last_seen", lastSeen).Error
}

func (repo *AddressRepositoryImpl) UpdateLastFailed(address *networking_models.Address, lastFailed *time.Time) error {
	return repo.db.Model(&db_models.AddressDB{}).
		Where("ip = ? AND port = ?", address.Ip.String(), address.Port).
		Update("last_failed", lastFailed).Error
}

func (repo *AddressRepositoryImpl) GetAddresses(limit int, excludedAddresses []*networking_models.Address) ([]*networking_models.Address, error) {
	whereClauses := []string{}
	args := make([]any, 0)

	if len(excludedAddresses) > 0 {
		pairs := make([]string, len(excludedAddresses))
		for i, addr := range excludedAddresses {
			pairs[i] = "(ip != ? OR port != ?)"
			args = append(args, addr.Ip.String(), addr.Port)
		}

		whereClauses = append(whereClauses, strings.Join(pairs, " AND "))
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf(`
        WITH ranked AS (
            SELECT *,
                   COALESCE(strftime('%%s', last_seen), 0) -
                   COALESCE(strftime('%%s', last_failed), 0) AS score,
                   ROW_NUMBER() OVER (
                       ORDER BY COALESCE(strftime('%%s', last_seen), 0) -
                                COALESCE(strftime('%%s', last_failed), 0) DESC
                   ) AS row_num
            FROM addresses
            %s
        ),
        biased AS (
            SELECT *,
                   (1.0 / row_num) * ABS(RANDOM()) AS random_score
            FROM ranked
        )
        SELECT * FROM biased
        ORDER BY random_score DESC
        LIMIT ?;
    `, whereSQL)

	args = append(args, limit)

	var addressesDB []*db_models.AddressDB
	if err := repo.db.Raw(query, args...).Scan(&addressesDB).Error; err != nil {
		return nil, err
	}

	addresses := make([]*networking_models.Address, len(addressesDB))
	for i, addressDB := range addressesDB {
		addresses[i] = mapping.AddressDBToAddress(addressDB)
	}

	return addresses, nil
}

func (repo *AddressRepositoryImpl) GetAddressesPaged(offset int, pageSize int, excludedAddresses []*networking_models.Address) ([]*db_models.AddressDB, int64, error) {
	query := repo.db.Model(&db_models.AddressDB{})

	for _, addr := range excludedAddresses {
		query = query.Where("NOT (ip = ? AND port = ?)", addr.Ip.String(), addr.Port)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var addressesDB []*db_models.AddressDB
	if err := query.Order("last_seen DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&addressesDB).Error; err != nil {
		return nil, 0, err
	}

	return addressesDB, total, nil
}
