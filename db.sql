CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50),
    email VARCHAR(50) UNIQUE,
    created_at TIMESTAMPTZ DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'America/Bogota')
);

CREATE TYPE option AS (
    id INT,
    variant_id INT,
    option VARCHAR(50)
);

CREATE TYPE items AS (
    sku VARCHAR,
    quantity INT
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    short_description VARCHAR(255),
    images TEXT[],
    created_at TIMESTAMPTZ DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'America/Bogota'),
    updated_at TIMESTAMPTZ DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'America/Bogota')
);

CREATE TABLE variants (
    id SERIAL PRIMARY KEY,
    label VARCHAR(50) NOT NULL,
    product_id INT NOT NULL,
    options option[],
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    UNIQUE(product_id, label)
);

CREATE TYPE currency AS ENUM ('USD', 'COP');
CREATE TABLE combinations (
    sku VARCHAR(255) UNIQUE PRIMARY KEY,
    price DECIMAL(15, 4) NOT NULL,
    currency currency NOT NULL,
    product_id INT NOT NULL,
    stock INT DEFAULT 0,
    options option[],
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

CREATE TABLE carts (
    id SERIAL PRIMARY KEY,
    user_id INT UNIQUE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE cart_items (
    cart_id INT NOT NULL,
    sku VARCHAR(255) NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'America/Bogota'),
    FOREIGN KEY (cart_id) REFERENCES carts(id) ON DELETE CASCADE,
    FOREIGN KEY (sku) REFERENCES combinations(sku) ON DELETE CASCADE,
    UNIQUE (sku, product_id, cart_id)
);

CREATE TYPE order_status AS ENUM ('COMPLETED', 'PENDING', 'PARTIALLY_REFUNDED', 'DECLINED', 'REFUNDED', 'FAILED');
CREATE TYPE payment_provider AS ENUM ('paypal');

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    order_id VARCHAR(255) NOT NULL,
    capture_ids TEXT[],
    reference_ids TEXT[],
    user_id INT NOT NULL,
    cart_items items[] NOT NULL,
    payer_name VARCHAR(255),
    payer_email VARCHAR(255),
    payer_id VARCHAR(255) NOT NULL,
    currency currency NOT NULL,
    total DECIMAL(15, 4) NOT NULL,
    status order_status DEFAULT 'PENDING',
    payment_provider payment_provider NOT NULL,
    created_at TIMESTAMPTZ DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'America/Bogota'),
    updated_at TIMESTAMPTZ DEFAULT (CURRENT_TIMESTAMP AT TIME ZONE 'America/Bogota'),
    FOREIGN KEY (user_id) REFERENCES users(id),
    UNIQUE(id, user_id, order_id)
);

CREATE TABLE favorites (
    id SERIAL PRIMARY KEY,
    user_id INT UNIQUE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE favorites_items (
    id SERIAL PRIMARY KEY,
    favorites_id INT,
    product_id INT,
    UNIQUE (favorites_id, product_id),
    FOREIGN KEY (favorites_id) REFERENCES favorites(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

CREATE OR REPLACE FUNCTION get_cart_items(
    in_user_id INT
) RETURNS TABLE (
    product_id INT,
    sku VARCHAR,
    quantity INT
) AS $$
DECLARE
    cart_id_var INT;
BEGIN
    SELECT id
    INTO cart_id_var
    FROM carts
    WHERE user_id = in_user_id;

    RETURN QUERY 
    SELECT ci.product_id, ci.sku, ci.quantity
    FROM cart_items AS ci
    WHERE ci.cart_id = cart_id_var;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION add_to_cart (
    in_user_id INT,
    in_sku VARCHAR,
    in_quantity INT,
    in_product_id INT
) RETURNS TABLE (
    cart_count_items INT
) AS $$
DECLARE
    cart_id_var INT;
    id_cart_items_var INT;
    item_quantity_var INT;
BEGIN
    SELECT id
    INTO cart_id_var
    FROM carts
    WHERE user_id = in_user_id
    LIMIT 1;

    IF EXISTS (
	SELECT 1
	FROM cart_items
	WHERE cart_id = cart_id_var AND sku = in_sku
    ) THEN
	UPDATE cart_items
	SET quantity = quantity + in_quantity
	WHERE cart_id = cart_id_var AND sku = in_sku
	RETURNING quantity INTO item_quantity_var;
    ELSE 
	INSERT INTO cart_items(cart_id, sku, product_id, quantity)
	VALUES (cart_id_var, in_sku, in_product_id, in_quantity)
	RETURNING quantity INTO item_quantity_var;
    END IF;

    IF item_quantity_var <= 0 THEN
        RAISE EXCEPTION 'Item quantity cannot be zero or less.';
    END IF;

    PERFORM FROM check_stock(item_quantity_var, in_sku);

    SELECT SUM(quantity)
    INTO cart_count_items
    FROM cart_items
    WHERE cart_items.cart_id = cart_id_var;

    RETURN QUERY
    SELECT cart_count_items;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION add_to_cart_with_item (
    in_user_id INT,
    in_sku VARCHAR,
    in_quantity INT,
    in_product_id INT
) RETURNS TABLE (
    product_id INT,
    sku VARCHAR,
    quantity INT,
    created_at TIMESTAMP,
    name VARCHAR,
    short_description TEXT,
    price DECIMAL,
    images TEXT[],
    options option[],
    cart_count_items INT
) AS $$
DECLARE
    cart_id_var INT;
    id_cart_items_var INT;
    quantity_var INT;
    item_quantity_var INT;
BEGIN
    SELECT id
    INTO cart_id_var
    FROM carts
    WHERE user_id = in_user_id
    LIMIT 1;

    IF EXISTS (
	SELECT 1
	FROM cart_items
	WHERE cart_id = cart_id_var AND sku = in_sku
    ) THEN
	UPDATE cart_items
	SET quantity = quantity + in_quantity
	WHERE cart_id = cart_id_var AND sku = in_sku
	RETURNING id, quantity INTO id_cart_items_var, item_quantity_var;
    ELSE 
	INSERT INTO cart_items(cart_id, sku, product_id, quantity)
	VALUES (cart_id_var, in_sku, in_product_id, in_quantity)
	RETURNING id, quantity INTO id_cart_items_var, item_quantity_var;
    END IF;

    IF item_quantity_var <= 0 THEN
        RAISE EXCEPTION 'Item quantity cannot be zero or less.';
    END IF;

    PERFORM FROM check_stock(item_quantity_var, in_sku);

    SELECT products.id, combinations.sku, cart_items.quantity,
	cart_items.created_at, products.name, products.short_description,
	combinations.price, products.images, combinations.options
    INTO product_id, sku, quantity, created_at, name,
	short_description, price, images, options
    FROM cart_items
    JOIN products ON products.id = cart_items.product_id
    JOIN combinations ON combinations.sku = cart_items.sku
    WHERE cart_items.id = id_cart_items_var
    ORDER BY cart_items.created_at DESC
    LIMIT 1;

    SELECT SUM(quantity)
    INTO cart_count_items
    FROM cart_items
    WHERE cart_items.cart_id = cart_id_var;

    RETURN QUERY
    SELECT product_id, sku, quantity, created_at, name,
	short_description, price, images, options, cart_count_items;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_cart_count (
    in_user_id INT,
    in_sku VARCHAR,
    in_quantity INT,
    in_product_id INT
) RETURNS TABLE (
    cart_count_items INT,
    product_count_items INT,
    total_product_balance DECIMAL,
    total_cart_balance DECIMAL
) AS $$
DEClARE
    cart_id_var INT;
    item_quantity_var INT;
BEGIN
    SELECT id
    INTO cart_id_var
    FROM carts
    WHERE user_id = in_user_id
    LIMIT 1;

    IF EXISTS (
	SELECT 1
	FROM cart_items
	WHERE cart_id = cart_id_var AND sku = in_sku
    ) THEN
	UPDATE cart_items
	SET quantity = in_quantity
	WHERE cart_id = cart_id_var AND sku = in_sku
	RETURNING quantity INTO product_count_items;
    ELSE 
	INSERT INTO cart_items(cart_id, sku, product_id, quantity)
	VALUES (cart_id_var, in_sku, in_product_id, in_quantity)
	RETURNING quantity INTO product_count_items;
    END IF;

    IF item_quantity_var <= 0 THEN
        RAISE EXCEPTION 'Item quantity cannot be zero or less.';
    END IF;

    PERFORM FROM check_stock(item_quantity_var, in_sku);

    SELECT SUM(quantity)
    INTO cart_count_items
    FROM cart_items
    WHERE cart_items.cart_id = cart_id_var;

    SELECT SUM(combinations.price * cart_items.quantity)
    INTO total_cart_balance
    FROM cart_items
    JOIN combinations ON cart_items.sku = combinations.sku
    WHERE cart_items.cart_id = cart_id_var;

    SELECT SUM(combinations.price * cart_items.quantity)
    INTO total_product_balance
    FROM cart_items
    JOIN combinations ON cart_items.sku = combinations.sku
    WHERE cart_items.cart_id = cart_id_var AND combinations.sku = in_sku
    LIMIT 1;

    RETURN QUERY
    SELECT cart_count_items, product_count_items, total_product_balance, total_cart_balance;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION cart_count_items (
    in_user_id INT
) RETURNS TABLE (
    cart_count_items INT
) AS $$
DEClARE
    cart_id_var INT;
BEGIN
    SELECT id
    INTO cart_id_var
    FROM carts
    WHERE user_id = in_user_id
    LIMIT 1;

    SELECT SUM(quantity)
    INTO cart_count_items
    FROM cart_items
    WHERE cart_items.cart_id = cart_id_var;

    IF cart_count_items IS NULL THEN
	cart_count_items := 0;
    END IF;

    RETURN QUERY
    SELECT cart_count_items;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION cart_count_items_with_total (
    in_user_id INT
) RETURNS TABLE (
    cart_count_items INT,
    total_cart_balance DECIMAL
) AS $$
DEClARE
    cart_id_var INT;
BEGIN
    SELECT id
    INTO cart_id_var
    FROM carts
    WHERE user_id = in_user_id
    LIMIT 1;

    SELECT SUM(quantity)
    INTO cart_count_items
    FROM cart_items
    WHERE cart_items.cart_id = cart_id_var;

    SELECT SUM(combinations.price * cart_items.quantity)
    INTO total_cart_balance
    FROM cart_items
    JOIN combinations ON cart_items.sku = combinations.sku
    WHERE cart_items.cart_id = cart_id_var;

    IF cart_count_items IS NULL THEN
	cart_count_items := 0;
    END IF;

    IF total_cart_balance IS NULL THEN
	total_cart_balance := 0;
    END IF;

    RETURN QUERY
    SELECT cart_count_items, total_cart_balance;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION delete_product_from_cart(
    in_user_id INT,
    in_sku VARCHAR
) RETURNS TABLE (
    cart_count_items INT,
    total_cart_balance DECIMAL
) AS $$
DECLARE
    cart_id_var INT;
BEGIN
    SELECT id
    INTO cart_id_var
    FROM carts
    WHERE user_id = in_user_id
    LIMIT 1;

    IF EXISTS (
	SELECT 1
	FROM cart_items
	WHERE cart_id = cart_id_var AND sku = in_sku
    ) THEN
	DELETE FROM cart_items
	WHERE cart_id = cart_id_var AND sku = in_sku;
    END IF;

    SELECT SUM(quantity)
    INTO cart_count_items
    FROM cart_items
    WHERE cart_id = cart_id_var;

    SELECT SUM(combinations.price * cart_items.quantity)
    INTO total_cart_balance
    FROM cart_items
    JOIN combinations ON cart_items.sku = combinations.sku
    WHERE cart_items.cart_id = cart_id_var;

    IF cart_count_items IS NULL THEN
	cart_count_items := 0;
    END IF;

    IF total_cart_balance IS NULL THEN 
	total_cart_balance := 0;
    END IF;

    RETURN QUERY
    SELECT cart_count_items, total_cart_balance;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION emptying_cart(
    in_user_id INT
) RETURNS VOID AS $$
DECLARE
    cart_id_var INT;
BEGIN
    SELECT id
    INTO cart_id_var
    FROM carts
    WHERE user_id = in_user_id
    LIMIT 1;

    DELETE
    FROM cart_items
    WHERE cart_id = cart_id_var;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_stock(
    in_items items[]
) RETURNS VOID AS $$
DECLARE
    item_var items;
    stock_var INT;
BEGIN
    FOREACH item_var IN ARRAY in_items
    LOOP
	UPDATE combinations 
	SET stock = stock - item_var.quantity
	WHERE sku = item_var.sku
	RETURNING stock INTO stock_var;
	
	IF stock_var < 0 THEN
	    RAISE EXCEPTION 'tried to store stock with invalid quantity below zero';
	END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;



CREATE OR REPLACE FUNCTION total_items(
    in_items items[]
) RETURNS TABLE (
    total_items_balance DECIMAL
) AS $$
DECLARE
    sum_holder_var DECIMAL := 0;
    item_var items;
BEGIN
    total_items_balance := 0;
    FOREACH item_var IN ARRAY in_items
    LOOP
	SELECT SUM(price * item_var.quantity)
	INTO sum_holder_var
	FROM combinations
	WHERE combinations.sku = item_var.sku
	LIMIT 1;

        total_items_balance := total_items_balance + sum_holder_var;
    END LOOP;

    RETURN QUERY
    SELECT total_items_balance;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION create_user(
    in_user_name VARCHAR,
    in_user_email VARCHAR
) RETURNS TABLE (
    user_id INT,
    out_created_at TIMESTAMPTZ,
    cart_id INT,
    favorites_id INT
) AS $$
BEGIN
    INSERT INTO users(name, email) VALUES (in_user_name, in_user_email) 
    RETURNING id, created_at INTO user_id, out_created_at;

    INSERT INTO carts(user_id) VALUES (user_id) 
    RETURNING id INTO cart_id;

    INSERT INTO favorites(user_id) VALUES (user_id) 
    RETURNING id INTO favorites_id;

    RETURN QUERY SELECT user_id, out_created_at, cart_id, favorites_id; 
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION get_core_user_data(
    in_user_id INT,
    in_user_email VARCHAR
) RETURNS TABLE (
    out_user_id INT,
    name VARCHAR,
    email VARCHAR,
    created_at TIMESTAMPTZ,
    cart_id INT,
    favorites_id INT
) AS $$
BEGIN
    SELECT u.id, u.name, u.email, u.created_at 
    INTO out_user_id, name, email, created_at
    FROM users u
    WHERE u.id = in_user_id OR u.email = in_user_email
    LIMIT 1;

    SELECT c.id
    INTO cart_id
    FROM carts c
    WHERE c.user_id = out_user_id
    LIMIT 1;

    SELECT f.id
    INTO favorites_id
    FROM favorites f
    WHERE f.user_id = out_user_id 
    LIMIT 1;

    RETURN QUERY
    SELECT out_user_id, name, email, created_at, cart_id, favorites_id;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION check_stock_from_items(
    in_items items[]
) RETURNS VOID AS $$
DECLARE
    item_var items;
    stock_var INT;
BEGIN
    FOREACH item_var IN ARRAY in_items
    LOOP
	SELECT stock
	INTO stock_var
	FROM combinations
	WHERE sku = item_var.sku
	LIMIT 1;

	IF item_var.quantity <= 0 THEN
	    RAISE EXCEPTION 'item quantity cant be equals or below zero.';
	END IF;

	IF stock_var < item_var.quantity THEN
	    RAISE EXCEPTION 'item quantity overpass stock.';
	END IF;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION check_stock_from_items_and_update_cart(
    in_items items[],
    in_user_id INT
) RETURNS TABLE (
    updated_cart BOOLEAN,
    error_message TEXT
) AS $$
DECLARE
    cart_id_var INT;
    item_var items;
    stock_var INT;
    count_error_item_quantity INT := 0;
    count_error_stock INT := 0;
BEGIN
    updated_cart := FALSE;
    error_message := '';

    SELECT id
    INTO cart_id_var
    FROM carts
    WHERE user_id = in_user_id
    LIMIT 1;

    FOREACH item_var IN ARRAY in_items
    LOOP
	SELECT stock
	INTO stock_var
	FROM combinations
	WHERE sku = item_var.sku
	LIMIT 1;

	IF item_var.quantity <= 0 THEN
	    updated_cart := TRUE;
	    count_error_item_quantity := count_error_item_quantity + 1;

	    IF EXISTS (
		SELECT 1
		FROM cart_items
		WHERE cart_id = cart_id_var AND sku = item_var.sku
	    ) THEN
		DELETE FROM cart_items
		WHERE cart_id = cart_id_var AND sku = item_var.sku;
	    END IF;
	END IF;

	IF stock_var < item_var.quantity THEN
	    updated_cart := TRUE;
	    count_error_stock := count_error_stock + 1;

	    UPDATE cart_items
	    SET quantity = stock_var
	    WHERE cart_id = cart_id_var AND sku = item_var.sku;
	END IF;
    END LOOP;

    IF count_error_stock > 0 OR count_error_item_quantity > 0 THEN
	error_message := FORMAT('encountered errors, error_stock: %s, error_item_quantity: %s', count_error_stock, count_error_item_quantity);
    END IF;

    RETURN QUERY
    SELECT updated_cart, error_message;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION check_stock (
    in_quantity INT,
    in_sku VARCHAR
) RETURNS VOID AS $$
DECLARE
    stock_var INT;
BEGIN
    SELECT stock
    INTO stock_var
    FROM combinations
    WHERE sku = in_sku
    LIMIT 1;

    IF in_quantity > stock_var THEN
	RAISE EXCEPTION 'item quantity overpass stock.';
    END IF;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION make_order(
    in_payment_provider payment_provider,
    in_user_id INT,
    in_cart_items items[],
    in_total DECIMAL,
    in_currency currency,
    in_order_id VARCHAR,
    in_payer_name VARCHAR,
    in_payer_email VARCHAR,
    in_payer_id VARCHAR,
    in_reference_ids TEXT[],
    in_capture_ids TEXT[]
) RETURNS TABLE (
    order_id INT
) AS $$
BEGIN
    INSERT INTO orders (payment_provider, user_id, cart_items,
	total, currency, order_id,
	payer_name, payer_email, payer_id,
	reference_ids, capture_ids)
    VALUES (in_payment_provider, in_user_id, in_cart_items,
	in_total, in_currency, in_order_id,
	in_payer_name, in_payer_email, in_payer_id,
	in_reference_ids, in_capture_ids)
    RETURNING id INTO order_id;

    RETURN QUERY
    SELECT order_id;
END;
$$ LANGUAGE plpgsql;


DROP FUNCTION IF EXISTS update_cart_count(INT, CHARVAR, INT, INT);
DROP FUNCTION IF EXISTS count_cart_items(INT);
DROP FUNCTION IF EXISTS cart_count_items(INT);
DROP FUNCTION IF EXISTS get_core_user_data(INT, VARCHAR);
DROP FUNCTION IF EXISTS create_user(VARCHAR, VARCHAR);
DROP FUNCTION IF EXISTS insert_variants(INT, variant[]);
DROP FUNCTION IF EXISTS insert_combinations(INT, VARCHAR, combination[]);
DROP FUNCTION IF EXISTS insert_product(VARCHAR, TEXT, VARCHAR, TEXT[]);
DROP FUNCTION IF EXISTS get_cart_items(INT);
DROP FUNCTION IF EXISTS add_to_cart(INT, VARCHAR, INT, INT);

DROP TYPE IF EXISTS combination CASCADE;
DROP TYPE IF EXISTS variant CASCADE;
DROP TYPE IF EXISTS option CASCADE;
DROP TYPE IF EXISTS items CASCADE;
DROP TYPE IF EXISTS order_status CASCADE;
DROP TYPE IF EXISTS currency CASCADE;
DROP TYPE IF EXISTS payment_provider CASCADE;

DROP TABLE IF EXISTS favorites_items CASCADE;
DROP TABLE IF EXISTS favorites CASCADE;
DROP TABLE IF EXISTS cart_items CASCADE;
DROP TABLE IF EXISTS variants CASCADE;
DROP TABLE IF EXISTS combinations CASCADE;
DROP TABLE IF EXISTS carts CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS orders CASCADE;
