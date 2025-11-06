package handlers

import (
	"Storm-Hunt/storm-backend/database"
	"Storm-Hunt/storm-backend/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func GetStorms(c *gin.Context) {
	var storms []models.Storm

	db := database.GetGormDB()
	if err := db.Find(&storms).Error; err != nil {
		log.Error().Err(err).Msg("Error finding storms")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Failed to find storms",
		})
		return
	}

	c.JSON(http.StatusOK, storms)
}

func GetStormByID(c *gin.Context) {
	var storm models.Storm

	db := database.GetGormDB()
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Error().Err(err).Msg("Error getting id")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid storm id",
		})
		return
	}

	if res := db.First(&storm, id); res == nil {
		log.Error().Msgf("Storm not found")
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Storm not found",
		})
		return
	}

	c.JSON(http.StatusOK, &storm)
}
