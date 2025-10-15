-- Migration: Add user constraints based on roles
-- For customers: email/phone must be unique within restaurant
-- For other roles: email/phone must be globally unique

-- Drop existing indexes if they exist
DROP INDEX IF EXISTS idx_users_email_restaurant_customer;
DROP INDEX IF EXISTS idx_users_phone_restaurant_customer;
DROP INDEX IF EXISTS idx_users_email_non_customer;
DROP INDEX IF EXISTS idx_users_phone_non_customer;

-- Create partial unique indexes for customers (email, restaurant_id) when role = 'customer'
CREATE UNIQUE INDEX idx_users_email_restaurant_customer 
ON users (email, restaurant_id) 
WHERE role = 'customer';

-- Create partial unique indexes for customers (phone, restaurant_id) when role = 'customer'
CREATE UNIQUE INDEX idx_users_phone_restaurant_customer 
ON users (phone, restaurant_id) 
WHERE role = 'customer';

-- Create unique indexes for non-customer roles (email globally unique)
CREATE UNIQUE INDEX idx_users_email_non_customer 
ON users (email) 
WHERE role != 'customer';

-- Create unique indexes for non-customer roles (phone globally unique)
CREATE UNIQUE INDEX idx_users_phone_non_customer 
ON users (phone) 
WHERE role != 'customer';

-- Add check constraint to ensure restaurant_id is provided for customers
ALTER TABLE users 
ADD CONSTRAINT chk_customer_restaurant_id 
CHECK (
    (role = 'customer' AND restaurant_id IS NOT NULL) OR 
    (role != 'customer')
);