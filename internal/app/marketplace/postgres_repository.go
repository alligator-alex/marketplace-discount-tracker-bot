package marketplace

import (
	"bot/internal/app/database"
	"bot/internal/app/helpers"
	"bot/internal/app/logger"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

type PostgresRepository struct {
	db     *database.Postgres
	logger logger.LoggerInterface
}

func NewPostgresRepository(db *database.Postgres, logger logger.LoggerInterface) PostgresRepository {
	return PostgresRepository{
		db:     db,
		logger: logger,
	}
}

// Find model by id.
func (r *PostgresRepository) FindById(id int) (Product, error) {
	sql := "SELECT * FROM products WHERE id = @id"

	args := pgx.NamedArgs{
		"id": id,
	}

	model, err := r.fetchModel(sql, args)
	if err != nil {
		return Product{}, err
	}

	return model, nil
}

// Find all models with page navigation.
func (r *PostgresRepository) FindAllPaginated(page int, perPage int) []Product {
	sql := "SELECT * FROM products LIMIT @limit OFFSET @offset"

	args := pgx.NamedArgs{
		"limit":  perPage,
		"offset": 0,
	}

	if page > 1 {
		args["offset"] = (page - 1) * perPage
	}

	return r.fetchModels(sql, args)
}

// Get count of all models.
func (r *PostgresRepository) GetCount() int {
	sql := "SELECT COUNT(*) FROM products"

	row := r.db.Connection.QueryRow(r.db.Context, sql)

	count := 0
	row.Scan(&count)

	return count
}

// Find outdated models with page navigation.
func (r *PostgresRepository) FindOutdatedPaginated(outdatedOffsetMinutes int, page int, perPage int) []Product {
	currentTime := time.Now()
	scrapedAtOutdated := currentTime.Add(time.Duration(-outdatedOffsetMinutes) * time.Minute)

	sql := "SELECT * FROM products WHERE scraped_at <= @scraped_at_outdated LIMIT @limit OFFSET @offset"

	args := pgx.NamedArgs{
		"scraped_at_outdated": helpers.TimeToDatabase(scrapedAtOutdated),
		"limit":               perPage,
		"offset":              0,
	}

	if page > 1 {
		args["offset"] = (page - 1) * perPage
	}

	return r.fetchModels(sql, args)
}

// Get count of outdated models.
func (r *PostgresRepository) GetCountOutdated(outdatedOffsetMinutes int) int {
	currentTime := time.Now()
	scrapedAtOutdated := currentTime.Add(time.Duration(-outdatedOffsetMinutes) * time.Minute)

	sql := "SELECT COUNT(*) FROM products WHERE scraped_at <= @scraped_at_outdated"

	args := pgx.NamedArgs{
		"scraped_at_outdated": helpers.TimeToDatabase(scrapedAtOutdated),
	}

	row := r.db.Connection.QueryRow(r.db.Context, sql, args)

	count := 0
	row.Scan(&count)

	return count
}

// Find model for user by URL.
func (r *PostgresRepository) FindForUserByUrl(telegramChatId int, telegramUserId int, url string) (Product, error) {
	sql := "SELECT * FROM products WHERE telegram_chat_id = @telegram_chat_id AND telegram_user_id = @telegram_user_id AND url = @url"

	args := pgx.NamedArgs{
		"telegram_chat_id": telegramChatId,
		"telegram_user_id": telegramUserId,
		"url":              url,
	}

	model, err := r.fetchModel(sql, args)
	if err != nil {
		return Product{}, err
	}

	return model, nil
}

// Find model for user by slug.
func (r *PostgresRepository) FindForUserBySlug(telegramChatId int, telegramUserId int, slug string) (Product, error) {
	sql := "SELECT * FROM products WHERE telegram_chat_id = @telegram_chat_id AND telegram_user_id = @telegram_user_id AND slug = @slug"

	args := pgx.NamedArgs{
		"telegram_chat_id": telegramChatId,
		"telegram_user_id": telegramUserId,
		"slug":             slug,
	}

	model, err := r.fetchModel(sql, args)
	if err != nil {
		return Product{}, err
	}

	return model, nil
}

// Delete model by id.
func (r *PostgresRepository) Delete(id int) bool {
	sql := "DELETE FROM products WHERE id = @id"

	args := pgx.NamedArgs{
		"id": id,
	}

	_, err := r.db.Connection.Exec(r.db.Context, sql, args)

	return err == nil
}

// Save model data.
func (r *PostgresRepository) Save(model Product) (Product, error) {
	transaction, err := r.db.Connection.Begin(r.db.Context)
	if err != nil {
		r.logger.Println("Unable to begin transaction:", err)
		return Product{}, err
	}

	defer transaction.Rollback(r.db.Context)

	if model.Exists() {
		model, err = r.updateModel(model)
	} else {
		model, err = r.insertModel(model)
	}

	if err == nil {
		transaction.Commit(r.db.Context)
	}

	return model, err
}

// Check if slug is unique.
func (r *PostgresRepository) IsUniqueSlug(slug string) bool {
	sql := "SELECT EXISTS (SELECT * FROM products WHERE slug = @slug)"

	args := pgx.NamedArgs{
		"slug": slug,
	}

	var exists bool

	err := r.db.Connection.QueryRow(r.db.Context, sql, args).Scan(&exists)
	if err != nil {
		return false
	}

	return !exists
}

// Execute SQL and fetch single model.
func (r *PostgresRepository) fetchModel(sql string, args pgx.NamedArgs) (Product, error) {
	model := Product{}

	rows, err := r.db.Connection.Query(r.db.Context, sql, args)
	if err != nil {
		return model, err
	}

	model, err = pgx.CollectExactlyOneRow(rows, r.rowToModel)
	if err == pgx.ErrNoRows {
		return model, nil
	}

	if err != nil {
		return model, err
	}

	return model, nil
}

// Execute SQL and fetch multiple models.
func (r *PostgresRepository) fetchModels(sql string, args pgx.NamedArgs) []Product {
	rows, err := r.db.Connection.Query(r.db.Context, sql, args)
	if err != nil {
		r.logger.Println("Unable to execute query:", err)
		os.Exit(0)
	}

	models, err := pgx.CollectRows[Product](rows, r.rowToModel)
	if err != nil {
		r.logger.Println("Unable to collect rows:", err)
		os.Exit(1)
	}

	return models
}

// Add new item to database.
func (r *PostgresRepository) insertModel(model Product) (Product, error) {
	currentTime := time.Now()

	sql := `INSERT INTO products (
		created_at, 
		updated_at, 
		scraped_at, 
		slug,
		telegram_chat_id, 
		telegram_user_id,
		url, 
		marketplace, 
		title, 
		threshold_price,
		current_price,
		out_of_stock
	) VALUES (
		@created_at, 
		@updated_at, 
		@scraped_at, 
		@slug,
		@telegram_chat_id, 
		@telegram_user_id,
		@url, 
		@marketplace, 
		@title, 
		@threshold_price,
		@current_price,
		@out_of_stock
	) RETURNING id`

	args := pgx.NamedArgs{
		"created_at":       currentTime,
		"updated_at":       currentTime,
		"scraped_at":       currentTime,
		"slug":             model.Slug,
		"telegram_chat_id": model.TelegramChatId,
		"telegram_user_id": model.TelegramUserId,
		"url":              model.Url,
		"marketplace":      model.Marketplace,
		"title":            model.Title,
		"threshold_price":  model.ThresholdPrice,
		"current_price":    model.CurrentPrice,
		"out_of_stock":     model.OutOfStock,
	}

	row := r.db.Connection.QueryRow(r.db.Context, sql, args)

	var id int
	err := row.Scan(&id)
	if err != nil {
		return Product{}, err
	}

	return r.FindById(id)
}

// Update existing item in database.
func (r *PostgresRepository) updateModel(model Product) (Product, error) {
	sql := `UPDATE products SET (
		updated_at,
		scraped_at,
		threshold_price,
		current_price,
		out_of_stock
	)=(
		@updated_at,
		@scraped_at,
		@threshold_price,
		@current_price,
		@out_of_stock
	) WHERE id=@id`

	args := pgx.NamedArgs{
		"id":              model.Id,
		"updated_at":      time.Now(),
		"scraped_at":      model.GetScrapedAt(),
		"threshold_price": model.GetThresholdPrice(),
		"current_price":   model.GetCurrentPrice(),
		"out_of_stock":    model.IsOutOfStock(),
	}

	_, err := r.db.Connection.Exec(r.db.Context, sql, args)
	if err != nil {
		return Product{}, err
	}

	return r.FindById(model.Id)
}

// Scan data from row to model.
func (r *PostgresRepository) rowToModel(row pgx.CollectableRow) (Product, error) {
	model := Product{}

	err := row.Scan(
		&model.Id,
		&model.CreatedAt,
		&model.UpdatedAt,
		&model.ScrapedAt,
		&model.Slug,
		&model.TelegramChatId,
		&model.TelegramUserId,
		&model.Url,
		&model.Marketplace,
		&model.Title,
		&model.ThresholdPrice,
		&model.CurrentPrice,
		&model.OutOfStock,
	)

	return model, err
}
