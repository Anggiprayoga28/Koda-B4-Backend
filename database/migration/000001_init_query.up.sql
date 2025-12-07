-- Active: 1763533642135@@165.22.110.38@5432@anggi-coffeshop
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(30) UNIQUE,
    password VARCHAR(255),
    role VARCHAR(25),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP 
);

CREATE TABLE user_profiles (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    full_name VARCHAR(255),
    phone VARCHAR(20),
    address TEXT,
    photo_url VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category_id INT NOT NULL,
    price INT NOT NULL,
    is_flash_sale BOOLEAN,
    is_favorite BOOLEAN,
    is_buy1get1 BOOLEAN,
    is_active BOOLEAN DEFAULT TRUE,
    stock INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_images (
    id SERIAL PRIMARY KEY,
    product_id INT NOT NULL,
    image_url VARCHAR(255) NOT NULL,
    is_primary BOOLEAN,
    display_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_reviews (
    id SERIAL PRIMARY KEY,
    product_id INT NOT NULL,
    user_id INT NOT NULL,
    rating INT NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review_text TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE promos (
    id SERIAL PRIMARY KEY,
    code VARCHAR(50) UNIQUE NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    discount_percentage INT NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE promo_products (
    id SERIAL PRIMARY KEY,
    promo_id INT NOT NULL,
    product_id INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE delivery_methods (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    base_fee INT NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE payment_methods (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tax_rates (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    rate_percentage DECIMAL(5,2) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_sizes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    price_adjustment INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_temperatures (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE order_status (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    display_order INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    order_number VARCHAR(50) UNIQUE NOT NULL,
    user_id INT NOT NULL,
    status_id INT NOT NULL,
    delivery_address TEXT NOT NULL,
    delivery_method_id INT NOT NULL,
    subtotal INT NOT NULL,
    delivery_fee INT NOT NULL,
    tax_amount INT NOT NULL,
    tax_rate_id INT,
    total INT NOT NULL,
    promo_id INT,
    payment_method_id INT NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    size_id INT,
    temperature_id INT,
    unit_price INT NOT NULL,
    is_flash_sale BOOLEAN,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE cart_items (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    size_id INT,
    temperature_id INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_variants (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE product_details (
    id SERIAL PRIMARY KEY,
    product_id INT NOT NULL,
    size_id INT,
    temperature_id INT,
    variant_id INT,
    price_adjustment INT DEFAULT 0,
    stock_adjustment INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (product_id) REFERENCES products(id),
    FOREIGN KEY (size_id) REFERENCES product_sizes(id),
    FOREIGN KEY (temperature_id) REFERENCES product_temperatures(id),
    FOREIGN KEY (variant_id) REFERENCES product_variants(id),
    UNIQUE(product_id, size_id, temperature_id, variant_id)
);
CREATE TABLE product_recommendations (
    id SERIAL PRIMARY KEY,
    product_detail_id INT NOT NULL,
    recommended_product_detail_id INT NOT NULL,
    recommendation_type VARCHAR(50) NOT NULL,
    priority INT DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (recommended_product_detail_id) REFERENCES product_details(id),
    UNIQUE(product_detail_id, recommended_product_detail_id)
);


ALTER TABLE product_images 
ADD CONSTRAINT check_image_position CHECK (display_order BETWEEN 1 AND 4);

ALTER TABLE product_images 
ADD CONSTRAINT unique_product_image_position UNIQUE (product_id, display_order);


ALTER TABLE user_profiles 
ADD CONSTRAINT fk_user_profiles_user 
FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE products 
ADD CONSTRAINT fk_products_category 
FOREIGN KEY (category_id) REFERENCES categories(id);

ALTER TABLE product_images 
ADD CONSTRAINT fk_product_images_product 
FOREIGN KEY (product_id) REFERENCES products(id);

ALTER TABLE product_reviews 
ADD CONSTRAINT fk_product_reviews_product 
FOREIGN KEY (product_id) REFERENCES products(id);

ALTER TABLE product_reviews 
ADD CONSTRAINT fk_product_reviews_user 
FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE promo_products 
ADD CONSTRAINT fk_promo_products_promo 
FOREIGN KEY (promo_id) REFERENCES promos(id);

ALTER TABLE promo_products 
ADD CONSTRAINT fk_promo_products_product 
FOREIGN KEY (product_id) REFERENCES products(id);

ALTER TABLE orders 
ADD CONSTRAINT fk_orders_user 
FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE orders 
ADD CONSTRAINT fk_orders_delivery_method 
FOREIGN KEY (delivery_method_id) REFERENCES delivery_methods(id);

ALTER TABLE orders 
ADD CONSTRAINT fk_orders_tax_rate 
FOREIGN KEY (tax_rate_id) REFERENCES tax_rates(id);

ALTER TABLE orders 
ADD CONSTRAINT fk_orders_promo 
FOREIGN KEY (promo_id) REFERENCES promos(id) ON DELETE SET NULL;

ALTER TABLE orders 
ADD CONSTRAINT fk_orders_payment_method 
FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id);

ALTER TABLE order_items 
ADD CONSTRAINT fk_order_items_order 
FOREIGN KEY (order_id) REFERENCES orders(id);

ALTER TABLE order_items 
ADD CONSTRAINT fk_order_items_product 
FOREIGN KEY (product_id) REFERENCES products(id);

ALTER TABLE order_items 
ADD CONSTRAINT fk_order_items_size 
FOREIGN KEY (size_id) REFERENCES product_sizes(id);

ALTER TABLE order_items 
ADD CONSTRAINT fk_order_items_temperature 
FOREIGN KEY (temperature_id) REFERENCES product_temperatures(id);

ALTER TABLE cart_items 
ADD CONSTRAINT fk_cart_items_user 
FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE cart_items 
ADD CONSTRAINT fk_cart_items_product 
FOREIGN KEY (product_id) REFERENCES products(id);

ALTER TABLE cart_items 
ADD CONSTRAINT fk_cart_items_size 
FOREIGN KEY (size_id) REFERENCES product_sizes(id);

ALTER TABLE cart_items 
ADD CONSTRAINT fk_cart_items_temperature 
FOREIGN KEY (temperature_id) REFERENCES product_temperatures(id);

ALTER TABLE products 
ADD COLUMN IF NOT EXISTS image_url VARCHAR(255);

ALTER TABLE product_temperatures 
ADD COLUMN price INT DEFAULT 0;

ALTER TABLE product_variants 
ADD COLUMN price INT DEFAULT 0;

ALTER TABLE cart_items 
ADD COLUMN variant_id INT;

ALTER TABLE cart_items 
ADD CONSTRAINT fk_cart_items_variant 
FOREIGN KEY (variant_id) REFERENCES product_variants(id);


ALTER TABLE orders 
ADD CONSTRAINT fk_orders_status 
FOREIGN KEY (status_id) REFERENCES order_status(id);

-- Add cloudinary_id column to products table for Cloudinary integration
ALTER TABLE products 
ADD COLUMN IF NOT EXISTS cloudinary_id VARCHAR(255);

-- Add index for better performance
CREATE INDEX IF NOT EXISTS idx_products_cloudinary_id ON products(cloudinary_id);

-- Create comment for documentation
COMMENT ON COLUMN products.cloudinary_id IS 'Cloudinary public ID for image management';

ALTER TABLE user_profiles 
ADD COLUMN cloudinary_public_id VARCHAR(255);

CREATE INDEX idx_user_profiles_cloudinary_public_id 
ON user_profiles(cloudinary_public_id);

-- Add UNIQUE constraint on user_id
ALTER TABLE user_profiles 
ADD CONSTRAINT user_profiles_user_id_key UNIQUE (user_id);

-- Add comment
COMMENT ON CONSTRAINT user_profiles_user_id_key ON user_profiles 
IS 'Ensures one profile per user for UPSERT operations';