package store_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"reflect"
	"shop/config"
	. "shop/services/store"

	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
	"time"
)

var pg PostgresStore

func variant() Variant {
	return Variant{
		Label: "Size",
		Options: []Option{
			{Id: 1, VariantId: 1, Option: "Small"},
			{Id: 2, VariantId: 1, Option: "Medium"},
			{Id: 3, VariantId: 1, Option: "Large"},
		},
	}
}

func TestInsertProducts(t *testing.T) {
	poolConn, err := pg.GetPool()
	if err != nil {
		log.Fatalf("init failed for testing: %v \n", err)
		return
	}
	tests := map[string]struct {
		input    Product
		expected Product
	}{
		`bothVoid`: {
			input:    Product{},
			expected: Product{},
		},
		`firstSample`: {
			input:    product(0),
			expected: product(2),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := insertDummiesProductV2(tt.input, poolConn)
			if err != nil {
				t.Errorf("input: %v, expected: %v,\nerr: %v", tt.input.Name, tt.expected.Name, err)
				return
			}
			if got.Id <= 0 {
				t.Errorf("got: %v, expected: %v, err: id cant be zero", got.Name, tt.expected.Name)
				return
			}
			tt.expected.CreatedAt = got.CreatedAt
			tt.expected.Id = got.Id
			queriedProduct, err := queryProductById(got.Id, poolConn)
			if err != nil {
				t.Errorf("failed to query product by id: %v, err: %v", got.Id, err)
				return
			}

			if !reflect.DeepEqual(queriedProduct, tt.expected) {
				t.Errorf("got: %v, expected: %v", queriedProduct, tt.expected)
				return
			}
			log.Println("works which is weird tho", queriedProduct, tt.expected)
		})
	}
}

func insertDummiesProductV2(product Product, pool *pgxpool.Pool) (Product, error) {
	ctx := context.Background()
	query := `
	INSERT INTO products_dummy_v2 (name, description, short_description, images)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at`
	err := pool.QueryRow(
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
	for _, combination := range product.Combinations {
		query := `
		INSERT INTO combinations_dummy_v2 (sku, price, stock, options, product_id)
		VALUES ($1, $2, $3, $4, $5)`
		ct, err := pool.Exec(ctx, query, combination.Sku, combination.Price, combination.Stock, combination.Options, product.Id)
		if err != nil {
			return Product{}, err
		}
		if ct.RowsAffected() <= 0 {
			return Product{}, fmt.Errorf("combination was not inserted")
		}
	}
	for _, variant := range product.Variants {
		query := `
		INSERT INTO variants_dummy_v2 (label, options, product_id)
		VALUES ($1, $2, $3)`
		ct, err := pool.Exec(ctx, query, variant.Label, variant.Options, product.Id)
		if err != nil {
			return Product{}, err
		}
		if ct.RowsAffected() <= 0 {
			return Product{}, fmt.Errorf("variant was not inserted")
		}
	}
	return product, nil
}

func queryProductById(id int, pool *pgxpool.Pool) (Product, error) {
	ctx := context.Background()
	var product Product
	query := `
	SELECT 
		p.id, p.name, p.description, p.short_description, p.images, p.created_at,
		(
			SELECT array_agg(v)
			FROM (
			SELECT v.label, v.options
			FROM variants_dummy_v2 AS v
			WHERE v.product_id = p.id
			) v
		) AS variants,
		(
			SELECT array_agg(c)
			FROM (
			SELECT c.id, c.sku, c.price, c.stock, c.options
			FROM combinations_dummy_v2 AS c
			WHERE c.product_id = p.id
			) c
		) AS combinations
	FROM products_dummy_v2 p
	WHERE p.id = $1;`

	err := pool.QueryRow(ctx, query, id).Scan(
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

func initDb() error {
	pool, err := pg.NewStore()
	if err != nil {
		return err
	}
	err = pg.SetNewPool(pool)
	if err != nil {
		return err
	}
	poolConn, err := pg.GetPool()
	if err != nil {
		return err
	}
	query := `
	CREATE TABLE products_dummy_v2 (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT NOT NULL,
		short_description VARCHAR(255),
		images TEXT[],  
		created_at TIMESTAMPTZ DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'America/Bogota')
	);

	CREATE TABLE variants_dummy_v2 (
		id SERIAL PRIMARY KEY,
		label VARCHAR(50) NOT NULL,
		product_id INT NOT NULL,
		options option[],
		FOREIGN KEY (product_id) REFERENCES products_dummy_v2(id) ON DELETE CASCADE,
		UNIQUE(product_id, label)
	);

	CREATE TABLE combinations_dummy_v2 (
		id SERIAL PRIMARY KEY,
		sku VARCHAR(255) NOT NULL,
		price DECIMAL(15, 4) NOT NULL,
		product_id INT NOT NULL,
		stock INT DEFAULT 0,
		options option[],
		FOREIGN KEY (product_id) REFERENCES products_dummy_v2(id) ON DELETE CASCADE
	);
	`
	_, err = poolConn.Exec(context.Background(), query)
	if err != nil {
		return err
	}
	return nil
}

func optionSlc(ind int) []Option {
	opts := []Option{
		{Id: 1, VariantId: 1, Option: "Small"},
		{Id: 2, VariantId: 1, Option: "Medium"},
		{Id: 3, VariantId: 1, Option: "Large"},
	}
	return opts[:ind]
}

func option() Option {
	return Option{Id: 0, VariantId: 1, Option: "Small"}
}

func insertProductT(product Product, pg Store) (Product, error) {
	p, err := pg.InsertProduct(context.Background(), product)
	if err != nil {
		return Product{}, err
	}
	return p, nil
}

func product(pp float64) Product {
	return Product{
		Id:               0,
		Name:             "T-Shirt",
		Description:      "A comfortable cotton t-shirt.",
		ShortDescription: "Soft and stylish t-shirt.",
		Images:           []string{"image1.jpg", "image2.jpg"},
		Variants: []Variant{
			{
				Label: "size",
				Options: []Option{
					{Id: 1, VariantId: 1, Option: "Small"},
					{Id: 2, VariantId: 1, Option: "Medium"},
					{Id: 3, VariantId: 1, Option: "Large"},
				},
			},
			{
				Label: "color",
				Options: []Option{
					{Id: 4, VariantId: 2, Option: "Red"},
					{Id: 5, VariantId: 2, Option: "Blue"},
					{Id: 6, VariantId: 2, Option: "Green"},
				},
			},
		},
		Combinations: []Combination{
			{
				Price:    pp,
				Currency: USD,
				Stock:    100,
				Options: []Option{
					{Id: 1, VariantId: 1, Option: "Small"},
					{Id: 4, VariantId: 2, Option: "Red"},
				},
			},
			{
				Price:    pp,
				Currency: USD,
				Stock:    50,
				Options: []Option{
					{Id: 2, VariantId: 1, Option: "Medium"},
					{Id: 5, VariantId: 2, Option: "Blue"},
				},
			},
			{
				Price:    pp,
				Currency: USD,
				Stock:    30,
				Options: []Option{
					{Id: 3, VariantId: 1, Option: "Large"},
					{Id: 6, VariantId: 2, Option: "Green"},
				},
			},
		},
		CreatedAt: time.Now(),
	}
}

func TestMain(m *testing.M) {
	defer pg.Close()
	err := config.LoadEnv()
	if err != nil {
		log.Fatalf("init failed for testing: %v \n", err)
	}
	err = initDb()
	if err != nil {
		log.Fatalf("init database failed for testing: %v \n", err)
	}

	exitCode := m.Run()
	cleanup()
	os.Exit(exitCode)
}

func cleanup() {
	poolConn, err := pg.GetPool()
	if err != nil {
		log.Fatalf("cleanup failed for testing: %v \n", err)
	}
	query := `
	DROP TABLE IF EXISTS combinations_dummy_v2 CASCADE;
	DROP TABLE IF EXISTS products_dummy_v2 CASCADE;
	DROP TABLE IF EXISTS variants_dummy_v2 CASCADE;
	`
	ct, err := poolConn.Exec(context.Background(), query)
	if err != nil {
		log.Fatalf("cleanup failed for testing: %v \n", err)
	}
	if ct.RowsAffected() < 0 {
		log.Fatalf("cleanup failed for testing, rows affected less than zero \n")
	}
}
