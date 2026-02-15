package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ParentDashboard returns info for the parent dashboard
func ParentDashboard(c *gin.Context) {
	// Example data, in a real app you might fetch child's grades, attendance, announcements, etc.
	c.JSON(http.StatusOK, gin.H{
		"name": "Parent User",
		"role": "parent",
		"children": []map[string]string{
			{"name": "Student 1", "grade": "A", "attendance": "95%"},
			{"name": "Student 2", "grade": "B+", "attendance": "90%"},
		},
	})
}
