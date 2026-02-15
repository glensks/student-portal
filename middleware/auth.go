package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("secret")

// ================= AUTH MIDDLEWARE =================
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 1️⃣ Get token from cookie
		if cookie, err := c.Cookie("jwt"); err == nil {
			tokenString = cookie
		}

		// 2️⃣ Or from Authorization header
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			log.Println("Missing JWT token")
			c.JSON(http.StatusForbidden, gin.H{"error": "missing token"})
			c.Abort()
			return
		}

		// 3️⃣ Parse & validate token
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			log.Printf("Invalid token: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			c.Abort()
			return
		}

		// 4️⃣ Extract role
		role, _ := claims["role"].(string)
		role = strings.ToLower(strings.TrimSpace(role))
		c.Set("role", role)

		// ✅ 5️⃣ Extract user_id (for ALL users - teacher, admin, student, etc.)
		if userID, ok := claims["user_id"].(float64); ok {
			c.Set("user_id", int(userID))
			log.Printf("User ID set: %d, Role: %s", int(userID), role)
		} else {
			log.Println("Warning: user_id not found in token")
		}

		// 6️⃣ For students, also set student_id if present
		if role == "student" {
			if sid, ok := claims["student_id"].(string); ok && sid != "" {
				c.Set("student_id", sid)
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "student_id missing in token"})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// ================= ROLE CHECK =================
func RoleOnly(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		r, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "role not set"})
			c.Abort()
			return
		}

		userRole := r.(string)

		for _, role := range allowedRoles {
			if strings.EqualFold(userRole, role) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden: role mismatch"})
		c.Abort()
	}
}
