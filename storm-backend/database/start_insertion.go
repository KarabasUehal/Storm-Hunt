package database

import (
	"Storm-Hunt/storm-backend/models"
	"fmt"

	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// Функция для вставки начальных данных о штормах
func StormInsertData(db *gorm.DB) error {
	var count_storms int64
	err := GORMDB.Model(&models.Storm{}).Count(&count_storms).Error // Проверка количества записей о штормах в БД
	if err != nil {
		log.Error().Err(err).Str("component", "mysql").Msg("[error] Failed to count storms table")
		return fmt.Errorf("failed to count storms table: %w", err)
	}

	if count_storms == 0 { // Если в БД нет записей, добавляются данные по ураганам

		insertData := `
    INSERT IGNORE INTO storms (name, region, date, wave_height, wind_speed, description) VALUES
    ('Nina', 'Atlantic', '1975-07-31', 44.4, 166, "One of the most powerful storm the world has ever seen"),
	('Ivan', 'Pacific', '1979-10-12', 24.0, 165, 'Largest tropical cyclone on record by diameter');
    `
		_, err = DB.Exec(insertData)
		if err != nil {
			log.Error().Err(err).Str("component", "mysql").Msg("Error to insert data into storms")
			return fmt.Errorf("failed to insert into storms: %w", err)
		}

		storm_map := MapDescription() // Создание отдельной таблицы с описаниями

		for i, v := range storm_map { // Добавление данных циклом по всей таблице
			var storm models.Storm
			if err := GORMDB.Where("name = ?", i).First(&storm).Error; err != nil {
				log.Error().Err(err).Msg("Error finding storm to update")
				return fmt.Errorf("failed to find %s in storms: %w", v, err)
			}
			storm.Description = v
			tx := GORMDB.Begin()
			if err := tx.Save(storm).Error; err != nil {
				tx.Rollback()
				log.Error().Err(err).Msg("Error to update storm description")
				return fmt.Errorf("failed to update storm description: %w", err)
			}
			tx.Commit()
		}
	}

	var count_storms_img int64
	err = GORMDB.Model(&models.StormImage{}).Count(&count_storms_img).Error // Проверка количества записей о изображениях в БД
	if err != nil {
		log.Error().Err(err).Str("component", "mysql").Msg("[error] Failed to count storm_images")
		return fmt.Errorf("failed to count storm_images: %w", err)
	}

	if count_storms_img == 0 { // Если в БД нет записей, добавляются изображения для штормов

		insertDataImg := `
    INSERT IGNORE INTO storm_images (storm_id, image_url, caption) VALUES
    (1, 'https://images.unsplash.com/photo-1458571037713-913d8b481dc6?ixlib=rb-4.0.3&auto=format&fit=crop&w=800&q=80', 'Satellite view on Nina'),
	(2, 'https://upload.wikimedia.org/wikipedia/commons/thumb/9/9d/Typhoon_Tip_1979.jpg/800px-Typhoon_Tip_1979.jpg', 'Eye of Typhoon Ivan');
    `
		_, err = DB.Exec(insertDataImg)
		if err != nil {
			log.Error().Err(err).Str("component", "mysql").Msg("Error to insert data into storm_images")
			return fmt.Errorf("failed to insert data into storm_images: %w", err)
		}
	}
	return nil
}

// Создание таблицы для добавления данных в цикле через GORM
func MapDescription() map[string]string {
	storm_map := make(map[string]string)

	storm_map["Nina"] = "Кратковременный и интенсивный супертайфун «Нина» случился в июле-августе 1975 года. Максимальная скорость урагана достигала 185 км/ч. «Нина» миновала Тайвань и обрушилась на прибрежный китайский город Хуалянь. Мощный ветер и сопровождавшие его ливни разрушили 62 плотины, в том числе крупнейшую Баньцяо на реке Жухэ. Причем Баньцяо была рассчитана на мощные наводнения, которые случаются раз в 1000 лет. Однако «Нина» принесла слишком много осадков, и створ дамбы не выдержал. Прорыв плотины привел к затоплению населенных пунктов ниже по течению реки. Высота волны достигала семи метров. В результате наводнения в прибрежной зоне океана погибли 26 000 человек, 100 000 человек умерли от последующего голода и болезней, а 230 000 человек — из-за прорыва плотины. Ущерб от «Нины» оценили в $1,2 млрд."
	storm_map["Ivan"] = "Ураган «Иван», также ураган «Айван» (англ. Hurricane Ivan) — 10-й по силе тропический циклон Атлантического океана за всю историю наблюдений. Это девятый проименованный тропический шторм и четвертый по силе ураган сезона 2004 года. Как типичный тропический циклон кабо-вердианского типа, он сформировался в начале сентября и достиг 5 категории по шкале Саффира-Симпсона. Во время прохождения по территории США ураган вызвал 117 смерчей."

	return storm_map
}
