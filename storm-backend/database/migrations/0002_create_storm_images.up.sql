CREATE TABLE IF NOT EXISTS `storm_images` (
    `id` INT AUTO_INCREMENT PRIMARY KEY,
    `storm_id` INT NOT NULL,
    `image_url` VARCHAR(1000) NOT NULL,
    `caption` VARCHAR(500),
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `deleted_at` TIMESTAMP NULL,
    INDEX `idx_storm_id` (`storm_id`),
    INDEX `idx_deleted` (`deleted_at`),
    FOREIGN KEY (`storm_id`) REFERENCES `storms`(`id`) ON DELETE CASCADE ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;