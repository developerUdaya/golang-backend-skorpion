-- Migration: Update OTP table to support optional restaurant_id for admin/staff OTPs

-- Remove NOT NULL constraint from restaurant_id in otps table
ALTER TABLE otps ALTER COLUMN restaurant_id DROP NOT NULL;

-- Add a comment to clarify the new behavior
COMMENT ON COLUMN otps.restaurant_id IS 'Restaurant ID for customer OTPs, NULL for admin/staff OTPs';