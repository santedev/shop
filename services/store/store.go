package store

import (
	"context"
	"errors"
	"fmt"
	"shop/config"
	"shop/gateaways"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var Pub Store

type PostgresStore struct {
	db *pgxpool.Pool
}

var ErrNoStock error = errors.New("ERROR: item quantity overpass stock. (SQLSTATE P0001)")

func (s *PostgresStore) SqlAddr() string {
	return fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		config.Envs.DBUser,
		config.Envs.DBPassword,
		config.Envs.DBHost,
		config.Envs.DBPort,
		config.Envs.DBName,
		config.Envs.DBsslMode,
	)
}

func (s *PostgresStore) SetNewPool(pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("pool is nil, cant set a nil pgxpool")
	}
	s.db = pool
	return nil
}

func (s *PostgresStore) GetPool() (*pgxpool.Pool, error) {
	if s.db == nil {
		return nil, fmt.Errorf("pg pool is nil")
	}
	return s.db, nil
}

func (s *PostgresStore) NewStore() (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(s.SqlAddr())
	if err != nil {
		return nil, fmt.Errorf("unable to parse database URL: %w", err)
	}

	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		dataTypeNames := []string{
			"option",
			"option[]",
			"items",
			"items[]",
			"currency",
			"order_status",
			"payment_provider",
		}
		for _, typeName := range dataTypeNames {
			dataType, err := conn.LoadType(ctx, typeName)
			if err != nil {
				return err
			}
			conn.TypeMap().RegisterType(dataType)
		}
		return nil
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)

	}
	err = pool.Ping(context.Background())
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func (s *PostgresStore) GetProductByName(ctx context.Context) (Product, error) {
	productName, ok := ctx.Value("productName").(string)
	if !ok {
		return Product{}, errors.New("product name was not found in the context")
	}
	if len(productName) <= 0 {
		return Product{}, errors.New("product name has an invalid length")
	}
	var product Product
	query := `
	SELECT id, name, description, short_description, images, variants, combinations, created_at
	FROM products
	WHERE name = $1`
	err := s.db.QueryRow(ctx, query, productName).Scan(
		&product.Id,
		&product.Name,
		&product.Description,
		&product.ShortDescription,
		&product.Images,
		&product.Variants,
		&product.Combinations,
		&product.CreatedAt)
	if err != nil {
		return Product{}, err
	}
	return product, nil
}

func (s *PostgresStore) GetProduct(ctx context.Context) (Product, error) {
	id, ok := ctx.Value("productId").(int)
	if !ok {
		return Product{}, errors.New("product id was not found in the context")
	}
	if id <= 0 {
		return Product{}, errors.New("product id is not valid, less than zero")
	}
	sku, ok := ctx.Value("sku").(Sku)
	if !ok {
		return Product{}, errors.New("sku was not found in the context")
	}
	if len(sku) <= 0 {
		return Product{}, errors.New("sku not valid cant be 0 len")
	}
	var product Product
	query := `
	SELECT 
		p.id, p.name, p.description, p.short_description, p.images, p.created_at,
		(
			SELECT array_agg(v)
			FROM (
			SELECT v.label, v.options
			FROM variants AS v
			WHERE v.product_id = p.id
			) v
		) AS variants,
		(
			SELECT array_agg(c)
			FROM (
			SELECT c.sku, c.price, c.currency, c.stock, c.options
			FROM combinations AS c
			WHERE c.product_id = p.id
			) c
		) AS combinations
	FROM products AS p
	WHERE p.id = $1;`

	err := s.db.QueryRow(ctx, query, id).Scan(
		&product.Id,
		&product.Name,
		&product.Description,
		&product.ShortDescription,
		&product.Images,
		&product.CreatedAt,
		&product.Variants,
		&product.Combinations,
	)
	if err != nil {
		return Product{}, err
	}
	return product, nil
}

func (s *PostgresStore) RemoveProduct(ctx context.Context) error {
	return nil
}

func (s *PostgresStore) InsertProduct(ctx context.Context, product Product) (Product, error) {
	query := `
	INSERT INTO products (name, description, short_description, images)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at`
	err := s.db.QueryRow(
		ctx,
		query,
		&product.Name,
		&product.Description,
		&product.ShortDescription,
		&product.Images,
	).
		Scan(&product.Id, &product.CreatedAt)
	if err != nil {
		return Product{}, err
	}
	pn := parseForSku(product.Name)
	for _, combination := range product.Combinations {
		var skuBld strings.Builder
		skuBld.WriteString(pn)
		skuBld.WriteString("-")
		for _, option := range combination.Options {
			skuBld.WriteString(parseForSku(option.Option))
			skuBld.WriteString("-")
		}
		sku, err := parsedSku(skuBld.String(), product.Id)
		if err != nil {
			return Product{}, err
		}
		combination.Sku = Sku(sku)
		query := `
		INSERT INTO combinations (sku, price, stock, currency, options, product_id)
		VALUES ($1, $2, $3, $4, $5, $6)`
		ct, err := s.db.Exec(ctx, query, combination.Sku, combination.Price, combination.Stock, combination.Currency, combination.Options, product.Id)
		if err != nil {
			return Product{}, err
		}
		if ct.RowsAffected() <= 0 {
			return Product{}, fmt.Errorf("combination was not inserted")
		}
	}
	for _, variant := range product.Variants {
		query := `
		INSERT INTO variants (label, options, product_id)
		VALUES ($1, $2, $3)`
		ct, err := s.db.Exec(ctx, query, variant.Label, variant.Options, product.Id)
		if err != nil {
			return Product{}, err
		}
		if ct.RowsAffected() <= 0 {
			return Product{}, fmt.Errorf("variant was not inserted")
		}
	}
	return product, nil
}

func (s *PostgresStore) UpdateProduct(ctx context.Context, product Product) (Product, error) {

	return Product{}, nil
}

func (s *PostgresStore) UpdateVariants(ctx context.Context, id int, variants []Variant) error {
	return nil
}

func (s *PostgresStore) UpdateCombinations(ctx context.Context, id int, combinations []Combination) error {
	return nil
}

func (s *PostgresStore) GetProducts(ctx context.Context, index int, limit int) ([]Product, error) {
	query := `
	SELECT 
		p.id, p.name, p.description, p.short_description, p.images, p.created_at,
		(
			SELECT array_agg(v)
			FROM (
				SELECT v.label, v.options
				FROM variants AS v
				WHERE v.product_id = p.id
			) v
		) AS variants,
		(
			SELECT array_agg(c)
			FROM (
				SELECT c.sku, c.price, c.currency, c.stock, c.options
				FROM combinations AS c
				WHERE c.product_id = p.id
			) c
		) AS combinations
	FROM products AS p
	WHERE p.id > $1
	LIMIT $2`
	rows, _ := s.db.Query(ctx, query, index, limit)
	products, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Product, error) {
		var product Product
		err := row.Scan(
			&product.Id,
			&product.Name,
			&product.Description,
			&product.ShortDescription,
			&product.Images,
			&product.CreatedAt,
			&product.Variants,
			&product.Combinations,
		)
		return product, err
	})
	if err != nil {
		return []Product{}, err
	}
	return products, nil
}

func (s *PostgresStore) AddToCart(ctx context.Context, userId int, sku Sku, quantity int) (int, error) {
	if userId <= 0 {
		return -1, errors.New("user id cant be equals or below zero")
	}
	if len(sku) <= 0 {
		return -1, errors.New("sku len cant be equals or below zero")
	}
	productId, err := sku.ProductId()
	if err != nil {
		return -1, err
	}
	query := `SELECT cart_count_items FROM add_to_cart($1, $2, $3, $4)`
	var cartCountItems int
	err = s.db.QueryRow(ctx, query, userId, sku, quantity, productId).Scan(&cartCountItems)
	if err != nil && strings.Contains(err.Error(), "item quantity overpass stock") {
		return -1, ErrNoStock
	}
	if err != nil {
		return -1, err
	}
	return cartCountItems, nil
}

func (s *PostgresStore) AddToCartWithItem(ctx context.Context, userId int, sku Sku, quantity int) (Items, int, error) {
	if userId <= 0 {
		return Items{}, -1, errors.New("user id cant be equals or below zero")
	}
	if len(sku) <= 0 {
		return Items{}, -1, errors.New("sku len cant be equals or below zero")
	}
	productId, err := sku.ProductId()
	if err != nil {
		return Items{}, -1, err
	}
	query := `
	SELECT product_id, sku, quantity, created_at,
	name, short_description, price, images, options, cart_count_items
	FROM add_to_cart_with_item($1, $2, $3, $4)
	`
	var (
		cartCountItems int
		items          Items
		combination    Combination
	)
	err = s.db.QueryRow(ctx, query, userId, sku, quantity, productId).Scan(
		&items.Id, &combination.Sku, &items.Quantity, &items.CreatedAt, &items.Name,
		&items.ShortDescription, &combination.Price, &items.Images, &combination.Options,
		&cartCountItems,
	)
	if err != nil {
		return Items{}, -1, err
	}
	items.Comb = combination
	return items, cartCountItems, nil
}

func (s *PostgresStore) UpdateCartCount(ctx context.Context, userId int, sku Sku, quantity int) (count, error) {
	if userId <= 0 {
		return count{}, errors.New("user id cant be equals or below zero")
	}
	if len(sku) <= 0 {
		return count{}, errors.New("sku len cant be equals or below zero")
	}
	productId, err := sku.ProductId()
	if err != nil {
		return count{}, err
	}
	query := `
	SELECT cart_count_items, product_count_items,
		total_product_balance, total_cart_balance
	FROM update_cart_count($1, $2, $3, $4)
	`
	var cartCounter count
	err = s.db.QueryRow(ctx, query, userId, string(sku), quantity, productId).Scan(
		&cartCounter.CartCount,
		&cartCounter.ProductCount,
		&cartCounter.ProductBalance,
		&cartCounter.CartBalance,
	)
	if err != nil {
		return count{}, err
	}
	return cartCounter, nil
}

func (s *PostgresStore) CartCountItems(ctx context.Context, userId int) (int, error) {
	if userId <= 0 {
		return -1, errors.New("user id cant be equals or below zero")
	}
	query := `
	SELECT cart_count_items FROM cart_count_items($1)
	`
	var countCart int
	err := s.db.QueryRow(ctx, query, userId).Scan(&countCart)
	if err != nil {
		return -1, err
	}
	return countCart, nil
}

func (s *PostgresStore) CartCountItemsWithTotal(ctx context.Context, userId int) (int, float64, error) {
	if userId <= 0 {
		return -1, -1, errors.New("user id cant be equals or below zero")
	}
	query := `
	SELECT cart_count_items, total_cart_balance FROM cart_count_items_with_total($1)
	`
	var (
		cartCount   int
		cartBalance float64
	)
	err := s.db.QueryRow(ctx, query, userId).Scan(&cartCount, &cartBalance)
	if err != nil {
		return -1, -1, err
	}
	return cartCount, cartBalance, nil
}
func (s *PostgresStore) CheckStockFromItemsAndUpdateCart(ctx context.Context, userId int, items []OrderItems) (bool, error) {
	if len(items) <= 0 {
		return false, errors.New("items len cant be equals or below zero")
	}
	if userId <= 0 {
		return false, errors.New("user id cant be equals or below zero")
	}
	query := `
	SELECT updated_cart, error_message FROM check_stock_from_items_and_update_cart($1, $2)`
	var (
		updatedCart  bool
		errorMessage string
	)
	err := s.db.QueryRow(ctx, query, items, userId).Scan(&updatedCart, &errorMessage)
	if err != nil {
		return false, err
	}
	if len(errorMessage) > 0 {
		return updatedCart, errors.New(errorMessage)
	}
	return updatedCart, nil
}
func (s *PostgresStore) CheckStockFromItems(ctx context.Context, items []OrderItems) error {
	if len(items) <= 0 {
		return errors.New("items len cant be equals or below zero")
	}
	query := `
	SELECT FROM check_stock_from_items($1)
	`
	_, err := s.db.Exec(ctx, query, items)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStore) UpdateStock(ctx context.Context, items []OrderItems) error {
	if len(items) <= 0 {
		return errors.New("items len cant be equals or below zero")
	}
	query := `
	SELECT FROM update_stock($1)
	`
	ct, err := s.db.Exec(ctx, query, items)
	if err != nil {
		return err
	}
	if ct.RowsAffected() <= 0 {
		return errors.New("something went wrong, RowsAffected not equals 0")
	}
	return nil
}

func (s *PostgresStore) RemoveProductFromCart(ctx context.Context, userId int, sku Sku) (count, error) {
	if userId <= 0 {
		return count{}, errors.New("user id cant be equals or below zero")
	}
	if len(sku) <= 0 {
		return count{}, errors.New("sku len cant be equals or below zero")
	}
	query := `
	SELECT cart_count_items, total_cart_balance FROM delete_product_from_cart($1,$2)
	`
	counter := count{}
	err := s.db.QueryRow(ctx, query, userId, string(sku)).Scan(&counter.CartCount, &counter.CartBalance)
	if err != nil {
		return count{}, err
	}
	return counter, nil
}

func (s *PostgresStore) GetCart(ctx context.Context, userId int) ([]Items, error) {
	if userId <= 0 {
		return []Items{}, errors.New("user id cant be equals or below zero")
	}
	query := `
	SELECT products.id, combinations.sku, cart_items.quantity, created_at,
		products.name, products.description, products.short_description,
		combinations.price, products.images, combinations.options
	FROM get_cart_items($1) AS cart_items
	JOIN products ON products.id = cart_items.product_id
	JOIN combinations ON combinations.sku = cart_items.sku
	ORDER BY created_at DESC
	`
	rows, _ := s.db.Query(ctx, query, userId)
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Items, error) {
		var item Items
		var combination Combination
		err := row.Scan(
			&item.Id,
			&combination.Sku,
			&item.Quantity,
			&item.CreatedAt,
			&item.Name,
			&item.Description,
			&item.ShortDescription,
			&combination.Price,
			&item.Images,
			&combination.Options,
		)
		if err != nil {
			return Items{}, err
		}
		item.Comb = combination
		return item, nil
	})
	if err != nil {
		return []Items{}, err
	}
	return items, nil
}

func (s *PostgresStore) EmptyingCart(ctx context.Context, userId int) error {
	if userId <= 0 {
		return errors.New("user id cant be zero or below")
	}
	query := `
	SELECT FROM emptying_cart($1)
	`
	ct, err := s.db.Exec(ctx, query, userId)
	if err != nil {
		return err
	}
	if ct.RowsAffected() <= 0 {
		return errors.New("there were no rows deleted")
	}
	return nil
}

func (s *PostgresStore) MakeOrder(ctx context.Context, paymentProvider gateaways.PaymentProvider, userId int, cartItems []OrderItems, total float64, currency currency, orderId, payerName, payerEmail, payerId string, referenceIds, captureIds []string) (int, error) {
	if err := paymentProvider.Valid(); err != nil {
		return -1, err
	}
	if err := currency.Valid(); err != nil {
		return -1, err
	}
	if len(cartItems) <= 0 {
		return -1, errors.New("len of items cant be zero or below")
	}
	if len(payerId) <= 0 {
		return -1, errors.New("len of payer id cant be zero or below")
	}
	if userId <= 0 {
		return -1, errors.New("user id cant be zero or below")
	}
	if total <= 0 {
		return -1, errors.New("total cant be zero or below")
	}

	query := `
	SELECT order_id FROM make_order($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	var id int
	err := s.db.QueryRow(ctx, query, paymentProvider, userId, cartItems, total, currency, orderId, payerName, payerEmail, payerId, referenceIds, captureIds).Scan(&id)
	if err != nil {
		return -1, err
	}
	return id, nil
}

func (s *PostgresStore) TotalItems(ctx context.Context, items []OrderItems) (float64, error) {
	if len(items) <= 0 {
		return -1, errors.New("len of items cant be zero or below")
	}
	query := `
	SELECT total_items_balance FROM total_items($1)
	`
	var total_items float64
	err := s.db.QueryRow(ctx, query, items).Scan(&total_items)
	if err != nil {
		return -1, err
	}
	return total_items, nil
}

func (s *PostgresStore) RestoreUser(ctx context.Context) (User, error) {
	user, ok := ctx.Value("user").(User)
	if !ok {
		return User{}, errors.New("cannot restore user, has not user struct. bad interface or void value")
	}
	validUid := user.Id > 0
	validEmail := len(user.Email) > 0
	if !validEmail && !validUid {
		return User{}, errors.New("there is no valid user context values to restore user info")
	}
	query := `
	SELECT out_user_id, name, email,
		created_at, cart_id, favorites_id
	FROM get_core_user_data($1,$2)`
	err := s.db.QueryRow(ctx, query, user.Id, user.Email).
		Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt, &user.CartId, &user.FavoritesId)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) NewUser(ctx context.Context, user User) (User, error) {
	if len(user.Name) == 0 {
		return User{}, errors.New("invalid length for username, cant set user to database")
	}
	if len(user.Email) == 0 {
		return User{}, errors.New("invalid length for user email, cant set user to database")
	}
	query := `
		SELECT user_id, out_created_at, cart_id, favorites_id
		FROM create_user($1, $2)
	`
	err := s.db.QueryRow(ctx, query, user.Name, user.Email).
		Scan(&user.Id, &user.CreatedAt, &user.CartId, &user.FavoritesId)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) GetUser(ctx context.Context) (User, error) {
	user, ok := ctx.Value("user").(User)
	if !ok {
		return User{}, errors.New("cannot restore user, has not user struct. bad interface or void value")
	}
	validUid := user.Id > 0
	validEmail := len(user.Email) > 0
	if !validEmail && !validUid {
		return User{}, errors.New("there is no valid user context values to restore user info")
	}
	query := `
	SELECT id, name, email, created_at
	FROM users
	WHERE id = $1 OR email = $2
	LIMIT 1
	`
	err := s.db.QueryRow(ctx, query, user.Id, user.Email).
		Scan(&user.Id, &user.Name, &user.Email, &user.CreatedAt)
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *PostgresStore) DeleteUser(ctx context.Context, uid string) error {
	id, err := strconv.Atoi(uid)
	if err != nil {
		return err
	}
	query := `DELETE FROM users WHERE id = $1`
	ct, err := s.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() != 1 {
		return errors.New("no row found to delete")
	}
	return nil
}

func (s *PostgresStore) CreateOrder(ctx context.Context, items []OrderItems) error {
	return nil
}

func (s *PostgresStore) Init() error {
	if Pub != nil {
		return fmt.Errorf("pub init function already called")
	}
	pool, err := s.NewStore()
	if err != nil {
		return err
	}
	s.db = pool
	Pub = s
	return err
}

func (s *PostgresStore) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func parsedSku(skuPreffix string, suffix int) (string, error) {
	if len(skuPreffix) <= 1 {
		return "", fmt.Errorf("preffix's len is less or equals than zero")
	}
	if suffix <= 0 {
		return "", fmt.Errorf("suffix cant be less or equals than zero")
	}
	if skuPreffix[len(skuPreffix)-1] == '-' {
		skuPreffix = skuPreffix[:len(skuPreffix)-1]
	}
	return skuPreffix + "-" + strconv.Itoa(suffix), nil
}

func parseForSku(option string) string {
	if len(option) <= 2 {
		return option
	}
	vocals := map[rune]struct{}{'A': {}, 'E': {}, 'I': {}, 'O': {}, 'U': {}}
	sku := make([]rune, 0, 2)
	for _, char := range option {
		if char >= 'a' && char <= 'z' {
			char -= 32
		}
		if _, exists := vocals[char]; exists || (char < 'A' || char > 'Z') {
			continue
		}
		sku = append(sku, char)
		if len(sku) >= 2 {
			break
		}
	}
	if len(sku) < 2 {
		return option[:2]
	}
	return string(sku)
}
