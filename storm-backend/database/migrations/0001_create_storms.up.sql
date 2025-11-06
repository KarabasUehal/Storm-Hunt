CREATE TABLE IF NOT EXISTS `storms` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(50) NOT NULL UNIQUE,
    `region` VARCHAR(100) NOT NULL,
    `date` DATE NOT NULL,
    `wave_height` DOUBLE PRECISION NOT NULL CHECK (wave_height >= 0),
    `wind_speed` SMALLINT NOT NULL CHECK (wind_speed >= 0),
    `description` TEXT NOT NULL,
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` TIMESTAMP NULL,
    INDEX `idx_region` (`region`),
    INDEX `idx_date` (`date`),
    INDEX `idx_deleted` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
    